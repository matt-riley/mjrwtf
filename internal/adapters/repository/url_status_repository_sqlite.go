//go:build ignore
// +build ignore

package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sqliterepo "github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/sqlite"
	"github.com/matt-riley/mjrwtf/internal/domain/urlstatus"
)

// SQLiteURLStatusRepository implements urlstatus.Repository for SQLite.
type SQLiteURLStatusRepository struct {
	db      *sql.DB
	queries *sqliterepo.Queries
}

func NewSQLiteURLStatusRepository(db *sql.DB) *SQLiteURLStatusRepository {
	return &SQLiteURLStatusRepository{
		db:      db,
		queries: sqliterepo.New(db),
	}
}

func (r *SQLiteURLStatusRepository) GetByURLID(ctx context.Context, urlID int64) (*urlstatus.URLStatus, error) {
	row, err := r.queries.GetURLStatusByURLID(ctx, urlID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
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

func (r *SQLiteURLStatusRepository) Upsert(ctx context.Context, status *urlstatus.URLStatus) error {
	if status == nil {
		return nil
	}
	return r.queries.UpsertURLStatus(ctx, sqliterepo.UpsertURLStatusParams{
		UrlID:            status.URLID,
		LastCheckedAt:    status.LastCheckedAt,
		LastStatusCode:   status.LastStatusCode,
		GoneAt:           status.GoneAt,
		ArchiveUrl:       status.ArchiveURL,
		ArchiveCheckedAt: status.ArchiveCheckedAt,
	})
}

func (r *SQLiteURLStatusRepository) ListDueForStatusCheck(ctx context.Context, aliveCutoff, goneCutoff time.Time, limit int) ([]*urlstatus.DueURL, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.queries.ListURLsDueForStatusCheck(ctx, sqliterepo.ListURLsDueForStatusCheckParams{
		LastCheckedAt:   &aliveCutoff,
		LastCheckedAt_2: &goneCutoff,
		Limit:           int64(limit),
	})
	if err != nil {
		return nil, err
	}

	out := make([]*urlstatus.DueURL, 0, len(rows))
	for _, row := range rows {
		out = append(out, &urlstatus.DueURL{
			URLID:            row.UrlID,
			ShortCode:        row.ShortCode,
			OriginalURL:      row.OriginalUrl,
			LastCheckedAt:    row.LastCheckedAt,
			LastStatusCode:   row.LastStatusCode,
			GoneAt:           row.GoneAt,
			ArchiveURL:       row.ArchiveUrl,
			ArchiveCheckedAt: row.ArchiveCheckedAt,
		})
	}
	return out, nil
}
