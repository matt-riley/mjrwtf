package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/sqlite"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// SQLiteURLRepository implements the URL repository for SQLite
type SQLiteURLRepository struct {
	urlRepositoryBase
	queries *sqliterepo.Queries
}

// NewSQLiteURLRepository creates a new SQLite URL repository
func NewSQLiteURLRepository(db *sql.DB) *SQLiteURLRepository {
	return &SQLiteURLRepository{
		urlRepositoryBase: urlRepositoryBase{db: db},
		queries:           sqliterepo.New(db),
	}
}

// Create creates a new shortened URL
func (r *SQLiteURLRepository) Create(ctx context.Context, u *url.URL) error {
	result, err := r.queries.CreateURL(ctx, sqliterepo.CreateURLParams{
		ShortCode:   u.ShortCode,
		OriginalUrl: u.OriginalURL,
		CreatedAt:   u.CreatedAt,
		CreatedBy:   u.CreatedBy,
	})

	if err != nil {
		return mapURLSQLError(err)
	}

	u.ID = result.ID
	return nil
}

// FindByShortCode retrieves a URL by its short code
func (r *SQLiteURLRepository) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	result, err := r.queries.FindURLByShortCode(ctx, shortCode)
	if err != nil {
		return nil, mapURLSQLError(err)
	}

	return &url.URL{
		ID:          result.ID,
		ShortCode:   result.ShortCode,
		OriginalURL: result.OriginalUrl,
		CreatedAt:   result.CreatedAt,
		CreatedBy:   result.CreatedBy,
	}, nil
}

// Delete removes a URL by its short code
func (r *SQLiteURLRepository) Delete(ctx context.Context, shortCode string) error {
	err := r.queries.DeleteURLByShortCode(ctx, shortCode)
	if err != nil {
		return mapURLSQLError(err)
	}

	return nil
}

// List retrieves URLs with optional filtering and pagination
func (r *SQLiteURLRepository) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	// Handle unlimited case
	if limit == 0 {
		limit = -1 // SQLite uses -1 for no limit
	}

	results, err := r.queries.ListURLs(ctx, sqliterepo.ListURLsParams{
		Column1:   createdBy,
		CreatedBy: createdBy,
		Limit:     int64(limit),
		Offset:    int64(offset),
	})

	if err != nil {
		return nil, mapURLSQLError(err)
	}

	urls := make([]*url.URL, len(results))
	for i, result := range results {
		urls[i] = &url.URL{
			ID:          result.ID,
			ShortCode:   result.ShortCode,
			OriginalURL: result.OriginalUrl,
			CreatedAt:   result.CreatedAt,
			CreatedBy:   result.CreatedBy,
		}
	}

	return urls, nil
}

// ListByCreatedByAndTimeRange retrieves URLs created by a specific user within a time range
func (r *SQLiteURLRepository) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	results, err := r.queries.ListURLsByCreatedByAndTimeRange(ctx, sqliterepo.ListURLsByCreatedByAndTimeRangeParams{
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
			ID:          result.ID,
			ShortCode:   result.ShortCode,
			OriginalURL: result.OriginalUrl,
			CreatedAt:   result.CreatedAt,
			CreatedBy:   result.CreatedBy,
		}
	}

	return urls, nil
}
