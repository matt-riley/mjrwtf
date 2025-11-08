package repository

import (
	"context"
	"database/sql"
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
func (r *PostgresURLRepository) Create(u *url.URL) error {
	ctx := context.Background()

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
func (r *PostgresURLRepository) FindByShortCode(shortCode string) (*url.URL, error) {
	ctx := context.Background()

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
func (r *PostgresURLRepository) Delete(shortCode string) error {
	ctx := context.Background()

	err := r.queries.DeleteURLByShortCode(ctx, shortCode)
	if err != nil {
		return mapURLSQLError(err)
	}

	return nil
}

// List retrieves URLs with optional filtering and pagination
func (r *PostgresURLRepository) List(createdBy string, limit, offset int) ([]*url.URL, error) {
	ctx := context.Background()

	// Handle unlimited case
	limitVal := int64(limit)
	if limit == 0 {
		limitVal = 2147483647 // Use max int32 for PostgreSQL
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
func (r *PostgresURLRepository) ListByCreatedByAndTimeRange(createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	ctx := context.Background()

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
