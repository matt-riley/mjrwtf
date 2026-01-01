package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a small HTTP client for the mjr.wtf API.
//
// It intentionally does not depend on server-side config packages so it can be used by
// the TUI (and other local tooling) without importing infrastructure.
type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	timeout    time.Duration
}

type Option func(*Client)

func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithTimeout sets a per-request timeout that is applied only when the provided context
// has no deadline.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("base url must include scheme and host")
	}

	c := &Client{
		baseURL:    u,
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Client) CreateURL(ctx context.Context, originalURL string) (*CreateURLResponse, error) {
	reqBody, err := json.Marshal(CreateURLRequest{OriginalURL: originalURL})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	u := c.resolve("/api/urls")
	req, cancel, err := c.newRequest(ctx, http.MethodPost, u, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer cancel()
	req.Header.Set("Content-Type", "application/json")

	var out CreateURLResponse
	if err := c.do(req, http.StatusCreated, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListURLs calls GET /api/urls.
//
// Note: passing limit=0 and/or offset=0 omits those query parameters so the server can apply
// its defaults (limit defaults to 20; offset defaults to 0).
func (c *Client) ListURLs(ctx context.Context, limit, offset int) (*ListURLsResponse, error) {
	u := c.resolve("/api/urls")
	q := u.Query()
	if limit != 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset != 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	u.RawQuery = q.Encode()

	req, cancel, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	defer cancel()

	var out ListURLsResponse
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteURL(ctx context.Context, shortCode string) error {
	u := c.resolve("/api/urls/" + url.PathEscape(shortCode))
	req, cancel, err := c.newRequest(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	defer cancel()

	return c.do(req, http.StatusNoContent, nil)
}

func (c *Client) GetAnalytics(ctx context.Context, shortCode string, startTime, endTime *time.Time) (*GetAnalyticsResponse, error) {
	u := c.resolve("/api/urls/" + url.PathEscape(shortCode) + "/analytics")
	q := u.Query()
	if startTime != nil {
		q.Set("start_time", startTime.UTC().Format(time.RFC3339))
	}
	if endTime != nil {
		q.Set("end_time", endTime.UTC().Format(time.RFC3339))
	}
	u.RawQuery = q.Encode()

	req, cancel, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	defer cancel()

	var out GetAnalyticsResponse
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) resolve(path string) *url.URL {
	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + path
	return &u
}

func (c *Client) newRequest(ctx context.Context, method string, u *url.URL, body io.Reader) (*http.Request, func(), error) {
	cancel := func() {}
	if c.timeout > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			ctx, cancel = context.WithTimeout(ctx, c.timeout)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		cancel()
		return nil, func() {}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, cancel, nil
}

func (c *Client) do(req *http.Request, expectedStatus int, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return decodeAPIError(resp)
	}

	if out == nil {
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			return fmt.Errorf("discard response body: %w", err)
		}
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	msg := ""
	var er ErrorResponse
	if err := json.Unmarshal(body, &er); err == nil {
		msg = er.Error
	}
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if msg == "" {
		msg = resp.Status
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    msg,
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if v := resp.Header.Get("Retry-After"); v != "" {
			if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
				apiErr.RetryAfter = time.Duration(secs) * time.Second
			}
		}
	}

	return apiErr
}
