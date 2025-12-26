package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/postgres"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// PostgresURLRepository implements the URL repository for PostgreSQL
type PostgresURLRepository struct {
	urlRepositoryBase
	queries *postgresrepo.Queries
}

// NewPostgresURLRepository creates a new PostgreSQL URL repository
func NewPostgresURLRepository(db *sql.DB) *PostgresURLRepository {
	return &PostgresURLRepository{
		urlRepositoryBase: urlRepositoryBase{db: db},
		queries:           postgresrepo.New(db),
	}
}

// Create creates a new shortened URL
func (r *PostgresURLRepository) Create(ctx context.Context, u *url.URL) error {
	result, err := r.queries.CreateURL(ctx, postgresrepo.CreateURLParams{
		ShortCode:   u.ShortCode,
		OriginalUrl: u.OriginalURL,
		CreatedAt:   u.CreatedAt,
		CreatedBy:   u.CreatedBy,
	})

	if err != nil {
		return mapURLSQLError(err)
	}

	u.ID = int64(result.ID)
	return nil
}

// FindByShortCode retrieves a URL by its short code
func (r *PostgresURLRepository) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	result, err := r.queries.FindURLByShortCode(ctx, shortCode)
	if err != nil {
		return nil, mapURLSQLError(err)
	}

	return &url.URL{
		ID:          int64(result.ID),
		ShortCode:   result.ShortCode,
		OriginalURL: result.OriginalUrl,
		CreatedAt:   result.CreatedAt,
		CreatedBy:   result.CreatedBy,
	}, nil
}

// Delete removes a URL by its short code
func (r *PostgresURLRepository) Delete(ctx context.Context, shortCode string) error {
	err := r.queries.DeleteURLByShortCode(ctx, shortCode)
	if err != nil {
		return mapURLSQLError(err)
	}

	return nil
}

// List retrieves URLs with optional filtering and pagination
func (r *PostgresURLRepository) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	// Handle unlimited case
	limitVal := int64(limit)
	if limit == 0 {
		limitVal = math.MaxInt32 // Use max int32 for PostgreSQL
	}

	// Validate that limit doesn't overflow int32
	if limitVal > math.MaxInt32 {
		return nil, fmt.Errorf("limit value %d exceeds maximum allowed value %d", limitVal, math.MaxInt32)
	}

	// Validate that offset doesn't overflow int32
	if offset > math.MaxInt32 || offset < 0 {
		return nil, fmt.Errorf("offset value %d is out of valid range (0 to %d)", offset, math.MaxInt32)
	}

	results, err := r.queries.ListURLs(ctx, postgresrepo.ListURLsParams{
		Column1:   createdBy,
		CreatedBy: createdBy,
		Limit:     int32(limitVal),
		Offset:    int32(offset),
	})

	if err != nil {
		return nil, mapURLSQLError(err)
	}

	urls := make([]*url.URL, len(results))
	for i, result := range results {
		urls[i] = &url.URL{
			ID:          int64(result.ID),
			ShortCode:   result.ShortCode,
			OriginalURL: result.OriginalUrl,
			CreatedAt:   result.CreatedAt,
			CreatedBy:   result.CreatedBy,
		}
	}

	return urls, nil
}

// ListByCreatedByAndTimeRange retrieves URLs created by a specific user within a time range
func (r *PostgresURLRepository) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	results, err := r.queries.ListURLsByCreatedByAndTimeRange(ctx, postgresrepo.ListURLsByCreatedByAndTimeRangeParams{
		CreatedBy:   createdBy,
		CreatedAt:   startTime,
		CreatedAt_2: endTime,
	})

	if err != nil {
		return nil, mapURLSQLError(err)
	}

	urls := make([]*url.URL, len(results))
	for i, result := range results {
		urls[i] = &url.URL{
			ID:          int64(result.ID),
			ShortCode:   result.ShortCode,
			OriginalURL: result.OriginalUrl,
			CreatedAt:   result.CreatedAt,
			CreatedBy:   result.CreatedBy,
		}
	}

	return urls, nil
}

// Count returns the total count of URLs for a specific user
func (r *PostgresURLRepository) Count(ctx context.Context, createdBy string) (int, error) {
	count, err := r.queries.CountURLsByCreatedBy(ctx, createdBy)

	if err != nil {
		return 0, mapURLSQLError(err)
	}

	return int(count), nil
}
