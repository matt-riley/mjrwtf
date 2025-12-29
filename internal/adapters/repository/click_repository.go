package repository

import (
	"database/sql"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
)

// ClickRepository is an interface implemented by this repo's SQLite repository adapters.
type ClickRepository interface {
	click.Repository
}

// clickRepositoryBase provides common functionality for Click repositories
type clickRepositoryBase struct {
	db *sql.DB
}

// mapClickSQLError maps SQL errors to domain errors
// Click repository currently has no specific domain errors for not found or duplicates,
// so we pass nil for both, which will wrap all errors as generic database errors
func mapClickSQLError(err error) error {
	return MapSQLError(err, nil, nil)
}

// stringToStringPtr converts string to *string, returning nil for empty string
func stringToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// stringToNullString converts string to sql.NullString
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
