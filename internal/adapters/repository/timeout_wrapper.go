package repository

import (
	"context"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// URLRepositoryWithTimeout wraps a URL repository and applies timeouts to all operations
type URLRepositoryWithTimeout struct {
	wrapped url.Repository
	timeout time.Duration
}

// NewURLRepositoryWithTimeout creates a URL repository wrapper that applies a timeout to all operations
func NewURLRepositoryWithTimeout(repo url.Repository, timeout time.Duration) *URLRepositoryWithTimeout {
	return &URLRepositoryWithTimeout{
		wrapped: repo,
		timeout: timeout,
	}
}

// Create creates a new shortened URL with a timeout
func (r *URLRepositoryWithTimeout) Create(ctx context.Context, u *url.URL) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.Create(ctx, u)
}

// FindByShortCode retrieves a URL by its short code with a timeout
func (r *URLRepositoryWithTimeout) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.FindByShortCode(ctx, shortCode)
}

// Delete removes a URL by its short code with a timeout
func (r *URLRepositoryWithTimeout) Delete(ctx context.Context, shortCode string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.Delete(ctx, shortCode)
}

// List retrieves URLs with optional filtering and pagination with a timeout
func (r *URLRepositoryWithTimeout) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.List(ctx, createdBy, limit, offset)
}

// ListByCreatedByAndTimeRange retrieves URLs created by a specific user within a time range with a timeout
func (r *URLRepositoryWithTimeout) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.ListByCreatedByAndTimeRange(ctx, createdBy, startTime, endTime)
}

// Count returns the total count of URLs for a specific user with a timeout
func (r *URLRepositoryWithTimeout) Count(ctx context.Context, createdBy string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.Count(ctx, createdBy)
}

// ClickRepositoryWithTimeout wraps a Click repository and applies timeouts to all operations
type ClickRepositoryWithTimeout struct {
	wrapped click.Repository
	timeout time.Duration
}

// NewClickRepositoryWithTimeout creates a Click repository wrapper that applies a timeout to all operations
func NewClickRepositoryWithTimeout(repo click.Repository, timeout time.Duration) *ClickRepositoryWithTimeout {
	return &ClickRepositoryWithTimeout{
		wrapped: repo,
		timeout: timeout,
	}
}

// Record records a new click event with a timeout
func (r *ClickRepositoryWithTimeout) Record(ctx context.Context, c *click.Click) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.Record(ctx, c)
}

// GetStatsByURL retrieves aggregate statistics for a specific URL with a timeout
func (r *ClickRepositoryWithTimeout) GetStatsByURL(ctx context.Context, urlID int64) (*click.Stats, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.GetStatsByURL(ctx, urlID)
}

// GetStatsByURLAndTimeRange retrieves statistics for a URL within a time range with a timeout
func (r *ClickRepositoryWithTimeout) GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.GetStatsByURLAndTimeRange(ctx, urlID, startTime, endTime)
}

// GetTotalClickCount returns the total number of clicks for a URL with a timeout
func (r *ClickRepositoryWithTimeout) GetTotalClickCount(ctx context.Context, urlID int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.GetTotalClickCount(ctx, urlID)
}

// GetClicksByCountry returns click counts grouped by country for a URL with a timeout
func (r *ClickRepositoryWithTimeout) GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.wrapped.GetClicksByCountry(ctx, urlID)
}
