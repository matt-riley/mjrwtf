package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/urlstatus"
	"github.com/rs/zerolog"
)

type URLStatusCheckerConfig struct {
	Enabled      bool
	PollInterval time.Duration

	// How often to re-check URLs that are not currently marked as gone.
	AliveRecheckInterval time.Duration
	// How often to re-check URLs that are marked as gone.
	GoneRecheckInterval time.Duration

	BatchSize   int
	Concurrency int

	ArchiveLookupEnabled   bool
	ArchiveRecheckInterval time.Duration
}

type URLStatusCheckerOption func(*URLStatusChecker)

func WithURLStatusCheckerHTTPClient(client *http.Client) URLStatusCheckerOption {
	return func(c *URLStatusChecker) {
		if client != nil {
			c.client = client
		}
	}
}

func WithURLStatusCheckerNow(now func() time.Time) URLStatusCheckerOption {
	return func(c *URLStatusChecker) {
		if now != nil {
			c.now = now
		}
	}
}

type URLStatusChecker struct {
	repo      urlstatus.Repository
	cfg       URLStatusCheckerConfig
	client    *http.Client
	logger    zerolog.Logger
	now       func() time.Time
	newTicker func(time.Duration) *time.Ticker

	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewURLStatusChecker(repo urlstatus.Repository, cfg URLStatusCheckerConfig, logger zerolog.Logger, opts ...URLStatusCheckerOption) *URLStatusChecker {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 5
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Minute
	}
	if cfg.AliveRecheckInterval <= 0 {
		cfg.AliveRecheckInterval = 6 * time.Hour
	}
	if cfg.GoneRecheckInterval <= 0 {
		cfg.GoneRecheckInterval = 24 * time.Hour
	}
	if cfg.ArchiveRecheckInterval <= 0 {
		cfg.ArchiveRecheckInterval = 7 * 24 * time.Hour
	}

	c := &URLStatusChecker{
		repo: repo,
		cfg:  cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Record the actual status code for the URL being checked (including 3xx).
				return http.ErrUseLastResponse
			},
		},
		logger:    logger,
		now:       time.Now,
		newTicker: time.NewTicker,
		stopCh:    make(chan struct{}),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if c.now == nil {
		c.now = time.Now
	}
	if c.newTicker == nil {
		c.newTicker = time.NewTicker
	}
	if c.client == nil {
		c.client = &http.Client{Timeout: 10 * time.Second}
	}

	return c
}

func (c *URLStatusChecker) Start(ctx context.Context) {
	if !c.cfg.Enabled {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		ticker := c.newTicker(c.cfg.PollInterval)
		defer ticker.Stop()

		// Run once on startup.
		c.runOnce(ctx)

		for {
			select {
			case <-c.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.runOnce(ctx)
			}
		}
	}()
}

func (c *URLStatusChecker) Shutdown() {
	c.stopOnce.Do(func() { close(c.stopCh) })
	c.wg.Wait()
}

// RunOnce runs a single check/persist pass; it is intended for tests and manual triggering.
func (c *URLStatusChecker) RunOnce(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.runOnce(ctx)
}

func (c *URLStatusChecker) runOnce(ctx context.Context) {
	now := c.now().UTC()
	aliveCutoff := now.Add(-c.cfg.AliveRecheckInterval)
	goneCutoff := now.Add(-c.cfg.GoneRecheckInterval)

	due, err := c.repo.ListDueForStatusCheck(ctx, aliveCutoff, goneCutoff, c.cfg.BatchSize)
	if err != nil {
		c.logger.Error().Err(err).Msg("url status checker: failed to list due URLs")
		return
	}
	if len(due) == 0 {
		return
	}

	sem := make(chan struct{}, c.cfg.Concurrency)
	var wg sync.WaitGroup

	for _, item := range due {
		item := item
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			c.checkAndPersist(ctx, now, item)
		}()
	}

	wg.Wait()
}

func (c *URLStatusChecker) checkAndPersist(ctx context.Context, now time.Time, item *urlstatus.DueURL) {
	code, err := c.fetchStatusCode(ctx, item.OriginalURL)
	if err != nil {
		c.logger.Debug().Err(err).Str("short_code", item.ShortCode).Msg("url status checker: check failed")
		st := &urlstatus.URLStatus{
			URLID:            item.URLID,
			LastCheckedAt:    &now,
			LastStatusCode:   nil,
			GoneAt:           item.GoneAt,
			ArchiveURL:       item.ArchiveURL,
			ArchiveCheckedAt: item.ArchiveCheckedAt,
		}
		if err := c.repo.Upsert(ctx, st); err != nil {
			c.logger.Error().Err(err).Str("short_code", item.ShortCode).Msg("url status checker: failed to persist status")
		}
		return
	}

	code64 := int64(code)
	isGone := urlstatus.IsGoneStatusCode(code)
	isServerError := code >= 500 && code < 600

	st := &urlstatus.URLStatus{
		URLID:          item.URLID,
		LastCheckedAt:  &now,
		LastStatusCode: &code64,
	}

	if isGone {
		if item.GoneAt != nil {
			st.GoneAt = item.GoneAt
		} else {
			st.GoneAt = &now
		}

		if c.cfg.ArchiveLookupEnabled {
			shouldLookup := item.ArchiveCheckedAt == nil || now.Sub(*item.ArchiveCheckedAt) >= c.cfg.ArchiveRecheckInterval
			if shouldLookup {
				archiveURL, lookupErr := c.lookupWayback(ctx, item.OriginalURL)
				if lookupErr != nil {
					c.logger.Debug().Err(lookupErr).Str("short_code", item.ShortCode).Msg("url status checker: archive lookup failed")
				}
				if archiveURL == "" {
					// Explicitly clear any previous archive URL when a lookup succeeds
					// but finds no archive, so stale values are not preserved.
					st.ArchiveURL = nil
				} else {
					st.ArchiveURL = &archiveURL
				}
				st.ArchiveCheckedAt = &now
			} else {
				st.ArchiveURL = item.ArchiveURL
				st.ArchiveCheckedAt = item.ArchiveCheckedAt
			}
		} else {
			st.ArchiveURL = item.ArchiveURL
			st.ArchiveCheckedAt = item.ArchiveCheckedAt
		}
	} else if isServerError {
		// Transient failure (e.g. 5xx) — preserve prior gone/archive state.
		st.GoneAt = item.GoneAt
		st.ArchiveURL = item.ArchiveURL
		st.ArchiveCheckedAt = item.ArchiveCheckedAt
	} else {
		// Destination is not gone (or recovered) — clear gone/archive fields.
		st.GoneAt = nil
		st.ArchiveURL = nil
		st.ArchiveCheckedAt = nil
	}

	if err := c.repo.Upsert(ctx, st); err != nil {
		c.logger.Error().Err(err).Str("short_code", item.ShortCode).Msg("url status checker: failed to persist status")
	}
}

func (c *URLStatusChecker) fetchStatusCode(ctx context.Context, rawURL string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "mjr.wtf-status-checker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Avoid reading entire bodies; we only care about status codes.
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)

	return resp.StatusCode, nil
}

type waybackAvailabilityResponse struct {
	ArchivedSnapshots struct {
		Closest struct {
			Available bool   `json:"available"`
			URL       string `json:"url"`
		} `json:"closest"`
	} `json:"archived_snapshots"`
}

func (c *URLStatusChecker) lookupWayback(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse("https://archive.org/wayback/available")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("url", rawURL)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mjr.wtf-status-checker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("wayback availability API returned %d", resp.StatusCode)
	}

	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	var out waybackAvailabilityResponse
	if err := dec.Decode(&out); err != nil {
		return "", err
	}

	if out.ArchivedSnapshots.Closest.Available && out.ArchivedSnapshots.Closest.URL != "" {
		return out.ArchivedSnapshots.Closest.URL, nil
	}

	return "", nil
}
