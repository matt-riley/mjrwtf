package click

import (
	"context"
	"time"
)

// Stats represents analytics statistics for a URL
type Stats struct {
	URLID      int64
	TotalCount int64
	ByCountry  map[string]int64
	ByReferrer map[string]int64
	ByDate     map[string]int64 // Date in YYYY-MM-DD format
}

// TimeRangeStats represents statistics for a specific time range
type TimeRangeStats struct {
	URLID      int64
	StartTime  time.Time
	EndTime    time.Time
	TotalCount int64
	ByCountry  map[string]int64
	ByReferrer map[string]int64
}

// Repository defines the interface for Click persistence operations
// Following hexagonal architecture, this interface is defined in the domain layer
// and implemented by adapters (e.g., PostgreSQL, SQLite)
type Repository interface {
	// Record records a new click event
	Record(ctx context.Context, click *Click) error

	// GetStatsByURL retrieves aggregate statistics for a specific URL
	GetStatsByURL(ctx context.Context, urlID int64) (*Stats, error)

	// GetStatsByURLAndTimeRange retrieves statistics for a URL within a time range
	GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*TimeRangeStats, error)

	// GetTotalClickCount returns the total number of clicks for a URL
	GetTotalClickCount(ctx context.Context, urlID int64) (int64, error)

	// GetClicksByCountry returns click counts grouped by country for a URL
	GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error)
}
