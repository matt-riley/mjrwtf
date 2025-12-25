package repository

import (
	"database/sql"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// URLRepository is an interface that can be implemented by both SQLite and PostgreSQL repositories
type URLRepository interface {
	url.Repository
}

// urlRepositoryBase provides common functionality for URL repositories
type urlRepositoryBase struct {
	db *sql.DB
}

// mapURLSQLError maps SQL errors to domain errors
func mapURLSQLError(err error) error {
	return MapSQLError(err, url.ErrURLNotFound, url.ErrDuplicateShortCode)
}
