package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/mattn/go-sqlite3"
)

// URLRepository is an interface that can be implemented by both SQLite and PostgreSQL repositories
type URLRepository interface {
	url.Repository
}

// urlRepositoryBase provides common functionality for URL repositories
type urlRepositoryBase struct {
	db *sql.DB
}

// isSQLiteUniqueConstraintError checks if the error is a SQLite unique constraint violation
func isSQLiteUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		// Error code 19 with extended code 2067 is SQLITE_CONSTRAINT_UNIQUE
		return sqliteErr.Code == sqlite3.ErrConstraint && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
	}
	return false
}

// isPostgresUniqueConstraintError checks if the error is a PostgreSQL unique constraint violation
func isPostgresUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// Error code 23505 is unique_violation
		return pqErr.Code == "23505"
	}
	return false
}

// mapSQLError maps SQL errors to domain errors
func mapURLSQLError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return url.ErrURLNotFound
	}

	if isSQLiteUniqueConstraintError(err) || isPostgresUniqueConstraintError(err) {
		return url.ErrDuplicateShortCode
	}

	return fmt.Errorf("database error: %w", err)
}
