package application

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/urlstatus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeURLStatusRepo struct {
	due     []*urlstatus.DueURL
	mu      sync.Mutex
	upserts []*urlstatus.URLStatus
}

func (r *fakeURLStatusRepo) GetByURLID(ctx context.Context, urlID int64) (*urlstatus.URLStatus, error) {
	return nil, nil
}

func (r *fakeURLStatusRepo) Upsert(ctx context.Context, status *urlstatus.URLStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *status
	r.upserts = append(r.upserts, &cp)
	return nil
}

func (r *fakeURLStatusRepo) ListDueForStatusCheck(ctx context.Context, aliveCutoff, goneCutoff time.Time, limit int) ([]*urlstatus.DueURL, error) {
	return r.due, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestURLStatusChecker_RunOnce_UpsertsAndClearsGoneOnAlive(t *testing.T) {
	fixedNow := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	goneAt := fixedNow.Add(-48 * time.Hour)
	archiveURL := "https://example.com/archive"
	archiveCheckedAt := fixedNow.Add(-24 * time.Hour)

	due := &urlstatus.DueURL{
		URLID:            123,
		ShortCode:        "abc",
		OriginalURL:      "https://example.com/alive",
		GoneAt:           &goneAt,
		ArchiveURL:       &archiveURL,
		ArchiveCheckedAt: &archiveCheckedAt,
	}

	repo := &fakeURLStatusRepo{due: []*urlstatus.DueURL{due}}

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", req.Method)
		}
		if req.URL.String() != due.OriginalURL {
			t.Fatalf("url = %s, want %s", req.URL.String(), due.OriginalURL)
		}
		if ua := req.Header.Get("User-Agent"); ua == "" {
			t.Fatal("expected User-Agent header")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	checker := NewURLStatusChecker(repo, URLStatusCheckerConfig{BatchSize: 10, Concurrency: 1}, zerolog.Nop(),
		WithURLStatusCheckerHTTPClient(client),
		WithURLStatusCheckerNow(func() time.Time { return fixedNow }),
	)

	checker.RunOnce(context.Background())

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.upserts, 1)
	got := repo.upserts[0]
	require.Equal(t, due.URLID, got.URLID)
	require.NotNil(t, got.LastCheckedAt)
	assert.True(t, got.LastCheckedAt.Equal(fixedNow))
	require.NotNil(t, got.LastStatusCode)
	assert.Equal(t, int64(http.StatusOK), *got.LastStatusCode)
	assert.Nil(t, got.GoneAt)
	assert.Nil(t, got.ArchiveURL)
	assert.Nil(t, got.ArchiveCheckedAt)
}

func TestURLStatusChecker_RunOnce_UpsertsGoneAndPreservesArchiveWhenLookupDisabled(t *testing.T) {
	fixedNow := time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC)
	archiveURL := "https://example.com/archive"
	archiveCheckedAt := fixedNow.Add(-24 * time.Hour)

	due := &urlstatus.DueURL{
		URLID:            456,
		ShortCode:        "def",
		OriginalURL:      "https://example.com/gone",
		ArchiveURL:       &archiveURL,
		ArchiveCheckedAt: &archiveCheckedAt,
	}

	repo := &fakeURLStatusRepo{due: []*urlstatus.DueURL{due}}

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	checker := NewURLStatusChecker(repo, URLStatusCheckerConfig{BatchSize: 10, Concurrency: 1}, zerolog.Nop(),
		WithURLStatusCheckerHTTPClient(client),
		WithURLStatusCheckerNow(func() time.Time { return fixedNow }),
	)

	checker.RunOnce(context.Background())

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.upserts, 1)
	got := repo.upserts[0]
	require.NotNil(t, got.GoneAt)
	assert.True(t, got.GoneAt.Equal(fixedNow))
	require.NotNil(t, got.LastStatusCode)
	assert.Equal(t, int64(http.StatusNotFound), *got.LastStatusCode)
	// When lookup is disabled, archive metadata is preserved on gone URLs.
	require.NotNil(t, got.ArchiveURL)
	assert.Equal(t, archiveURL, *got.ArchiveURL)
	require.NotNil(t, got.ArchiveCheckedAt)
	assert.True(t, got.ArchiveCheckedAt.Equal(archiveCheckedAt))
}

func TestURLStatusChecker_RunOnce_UpsertsOnFetchErrorWithoutNetwork(t *testing.T) {
	fixedNow := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	goneAt := fixedNow.Add(-72 * time.Hour)

	due := &urlstatus.DueURL{
		URLID:       789,
		ShortCode:   "ghi",
		OriginalURL: "https://example.com/error",
		GoneAt:      &goneAt,
	}

	repo := &fakeURLStatusRepo{due: []*urlstatus.DueURL{due}}

	expectedErr := errors.New("boom")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, expectedErr
	})}

	checker := NewURLStatusChecker(repo, URLStatusCheckerConfig{BatchSize: 10, Concurrency: 1}, zerolog.Nop(),
		WithURLStatusCheckerHTTPClient(client),
		WithURLStatusCheckerNow(func() time.Time { return fixedNow }),
	)

	checker.RunOnce(context.Background())

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.upserts, 1)
	got := repo.upserts[0]
	require.NotNil(t, got.LastCheckedAt)
	assert.True(t, got.LastCheckedAt.Equal(fixedNow))
	assert.Nil(t, got.LastStatusCode)
	require.NotNil(t, got.GoneAt)
	assert.True(t, got.GoneAt.Equal(goneAt))
}

func TestURLStatusChecker_RunOnce_PerformsArchiveLookupWhenEnabled(t *testing.T) {
	fixedNow := time.Date(2025, 4, 5, 6, 7, 8, 0, time.UTC)
	orig := "https://example.com/gone"
	wantArchive := "https://web.archive.org/web/20200101000000/https://example.com/gone"

	due := &urlstatus.DueURL{
		URLID:            999,
		ShortCode:        "zzz",
		OriginalURL:      orig,
		ArchiveCheckedAt: nil,
	}

	repo := &fakeURLStatusRepo{due: []*urlstatus.DueURL{due}}

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if ua := req.Header.Get("User-Agent"); ua == "" {
			t.Fatal("expected User-Agent header")
		}

		switch {
		case req.URL.String() == orig:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		case req.URL.Host == "archive.org" && req.URL.Path == "/wayback/available":
			if got := req.URL.Query().Get("url"); got != orig {
				t.Fatalf("wayback query url = %q, want %q", got, orig)
			}
			body := `{"archived_snapshots":{"closest":{"available":true,"url":"` + wantArchive + `"}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	})}

	checker := NewURLStatusChecker(repo, URLStatusCheckerConfig{BatchSize: 10, Concurrency: 1, ArchiveLookupEnabled: true}, zerolog.Nop(),
		WithURLStatusCheckerHTTPClient(client),
		WithURLStatusCheckerNow(func() time.Time { return fixedNow }),
	)

	checker.RunOnce(context.Background())

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.upserts, 1)
	got := repo.upserts[0]
	require.NotNil(t, got.ArchiveCheckedAt)
	assert.True(t, got.ArchiveCheckedAt.Equal(fixedNow))
	require.NotNil(t, got.ArchiveURL)
	assert.Equal(t, wantArchive, *got.ArchiveURL)
}
