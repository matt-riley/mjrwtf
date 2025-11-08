package repository

import (
	"database/sql"
	"fmt"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
)

// ClickRepository is an interface that can be implemented by both SQLite and PostgreSQL repositories
type ClickRepository interface {
	click.Repository
}

// clickRepositoryBase provides common functionality for Click repositories
type clickRepositoryBase struct {
	db *sql.DB
}

// mapClickSQLError maps SQL errors to domain errors
func mapClickSQLError(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("database error: %w", err)
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
