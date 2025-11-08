package repository

import (
	"database/sql"
	"errors"
	"fmt"

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

// isSQLiteUniqueConstraintError checks if the error is a SQLite unique constraint violation
func isSQLiteUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite error 19 is a constraint violation
	// SQLite error message contains "UNIQUE constraint failed"
	return err.Error() == "UNIQUE constraint failed: urls.short_code"
}

// isPostgresUniqueConstraintError checks if the error is a PostgreSQL unique constraint violation
func isPostgresUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL error code 23505 is unique_violation
	return err.Error() == "pq: duplicate key value violates unique constraint \"urls_short_code_key\""
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
