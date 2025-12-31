package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/sqlite"
	"github.com/matt-riley/mjrwtf/internal/domain/urlstatus"
)

// SQLiteURLStatusRepository implements the URLStatus repository for SQLite.
type SQLiteURLStatusRepository struct {
	urlStatusRepositoryBase
	queries *sqliterepo.Queries
}

// NewSQLiteURLStatusRepository creates a new SQLite URLStatus repository.
func NewSQLiteURLStatusRepository(db *sql.DB) *SQLiteURLStatusRepository {
	return &SQLiteURLStatusRepository{
		urlStatusRepositoryBase: urlStatusRepositoryBase{db: db},
		queries:                 sqliterepo.New(db),
	}
}

// GetByURLID retrieves URL status metadata for a URL.
// If no status row exists, it returns (nil, nil).
func (r *SQLiteURLStatusRepository) GetByURLID(ctx context.Context, urlID int64) (*urlstatus.URLStatus, error) {
	row, err := r.queries.GetURLStatusByURLID(ctx, urlID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &urlstatus.URLStatus{
		URLID:            row.UrlID,
		LastCheckedAt:    row.LastCheckedAt,
		LastStatusCode:   row.LastStatusCode,
		GoneAt:           row.GoneAt,
		ArchiveURL:       row.ArchiveUrl,
		ArchiveCheckedAt: row.ArchiveCheckedAt,
	}, nil
}

// Upsert inserts or updates URL status metadata.
func (r *SQLiteURLStatusRepository) Upsert(ctx context.Context, status *urlstatus.URLStatus) error {
	if status == nil {
		return fmt.Errorf("database error: urlstatus is nil")
	}

	err := r.queries.UpsertURLStatus(ctx, sqliterepo.UpsertURLStatusParams{
		UrlID:            status.URLID,
		LastCheckedAt:    status.LastCheckedAt,
		LastStatusCode:   status.LastStatusCode,
		GoneAt:           status.GoneAt,
		ArchiveUrl:       status.ArchiveURL,
		ArchiveCheckedAt: status.ArchiveCheckedAt,
	})
	if err != nil {
		return MapSQLError(err, nil, nil)
	}
	return nil
}

// ListDueForStatusCheck returns URLs that should be processed by the periodic status checker.
func (r *SQLiteURLStatusRepository) ListDueForStatusCheck(ctx context.Context, aliveCutoff, goneCutoff time.Time, limit int) ([]*urlstatus.DueURL, error) {
	if limit == 0 {
		limit = -1 // SQLite uses -1 for no limit
	}
	aliveCutoffPtr := &aliveCutoff
	goneCutoffPtr := &goneCutoff

	rows, err := r.queries.ListURLsDueForStatusCheck(ctx, sqliterepo.ListURLsDueForStatusCheckParams{
		LastCheckedAt:   aliveCutoffPtr,
		LastCheckedAt_2: goneCutoffPtr,
		Limit:           int64(limit),
	})
	if err != nil {
		return nil, MapSQLError(err, nil, nil)
	}

	out := make([]*urlstatus.DueURL, len(rows))
	for i, row := range rows {
		out[i] = &urlstatus.DueURL{
			URLID:            row.UrlID,
			ShortCode:        row.ShortCode,
			OriginalURL:      row.OriginalUrl,
			LastCheckedAt:    row.LastCheckedAt,
			LastStatusCode:   row.LastStatusCode,
			GoneAt:           row.GoneAt,
			ArchiveURL:       row.ArchiveUrl,
			ArchiveCheckedAt: row.ArchiveCheckedAt,
		}
	}

	return out, nil
}
