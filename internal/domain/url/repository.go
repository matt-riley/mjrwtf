package url

import "time"

// Repository defines the interface for URL persistence operations
// Following hexagonal architecture, this interface is defined in the domain layer
// and implemented by adapters (e.g., PostgreSQL, SQLite)
type Repository interface {
	// Create creates a new shortened URL
	// Returns ErrDuplicateShortCode if the short code already exists
	Create(url *URL) error

	// FindByShortCode retrieves a URL by its short code
	// Returns ErrURLNotFound if the URL doesn't exist
	FindByShortCode(shortCode string) (*URL, error)

	// Delete removes a URL by its short code
	// Returns ErrURLNotFound if the URL doesn't exist
	Delete(shortCode string) error

	// List retrieves URLs with optional filtering and pagination
	// createdBy: filter by creator (empty string means no filter)
	// limit: maximum number of results to return (0 means no limit)
	// offset: number of results to skip
	List(createdBy string, limit, offset int) ([]*URL, error)

	// ListByCreatedByAndTimeRange retrieves URLs created by a specific user within a time range
	ListByCreatedByAndTimeRange(createdBy string, startTime, endTime time.Time) ([]*URL, error)
}
