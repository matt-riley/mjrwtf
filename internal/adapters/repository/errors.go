package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
)

// IsSQLiteUniqueConstraintError checks if the error is a SQLite unique constraint violation
func IsSQLiteUniqueConstraintError(err error) bool {
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

// IsPostgresUniqueConstraintError checks if the error is a PostgreSQL unique constraint violation
func IsPostgresUniqueConstraintError(err error) bool {
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

// IsUniqueConstraintError checks if the error is a unique constraint violation for either SQLite or PostgreSQL
func IsUniqueConstraintError(err error) bool {
	return IsSQLiteUniqueConstraintError(err) || IsPostgresUniqueConstraintError(err)
}

// IsNoRowsError checks if the error is sql.ErrNoRows
func IsNoRowsError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// MapSQLError maps SQL errors to domain errors.
// It returns notFoundErr for sql.ErrNoRows, duplicateErr for unique constraint violations,
// and wraps other errors as generic database errors.
// If notFoundErr is nil, sql.ErrNoRows is wrapped as a generic database error.
// If duplicateErr is nil, unique constraint violations are wrapped as generic database errors.
func MapSQLError(err error, notFoundErr error, duplicateErr error) error {
	if err == nil {
		return nil
	}

	if notFoundErr != nil && IsNoRowsError(err) {
		return notFoundErr
	}

	if duplicateErr != nil && IsUniqueConstraintError(err) {
		return duplicateErr
	}

	return fmt.Errorf("database error: %w", err)
}
