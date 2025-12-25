package url

import (
	"context"
	"time"
)

// Repository defines the interface for URL persistence operations
// Following hexagonal architecture, this interface is defined in the domain layer
// and implemented by adapters (e.g., PostgreSQL, SQLite)
type Repository interface {
	// Create creates a new shortened URL
	// Returns ErrDuplicateShortCode if the short code already exists
	Create(ctx context.Context, url *URL) error

	// FindByShortCode retrieves a URL by its short code
	// Returns ErrURLNotFound if the URL doesn't exist
	FindByShortCode(ctx context.Context, shortCode string) (*URL, error)

	// Delete removes a URL by its short code
	// Returns ErrURLNotFound if the URL doesn't exist
	Delete(ctx context.Context, shortCode string) error

	// List retrieves URLs with optional filtering and pagination
	// createdBy: filter by creator (empty string means no filter)
	// limit: maximum number of results to return (0 means no limit)
	// offset: number of results to skip
	List(ctx context.Context, createdBy string, limit, offset int) ([]*URL, error)

	// ListByCreatedByAndTimeRange retrieves URLs created by a specific user within a time range
	ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*URL, error)

	// Count returns the total count of URLs for a specific user
	// createdBy: filter by creator (empty string means count all URLs)
	Count(ctx context.Context, createdBy string) (int, error)
}
