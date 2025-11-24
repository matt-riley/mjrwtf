package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/matt-riley/mjrwtf/internal/domain/session"
)

// SessionRepository is an interface that can be implemented by both SQLite and PostgreSQL repositories
type SessionRepository interface {
	session.Repository
}

// sessionRepositoryBase provides common functionality for session repositories
type sessionRepositoryBase struct {
	db *sql.DB
}

// mapSessionSQLError maps SQL errors to domain errors
func mapSessionSQLError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return session.ErrSessionNotFound
	}

	return fmt.Errorf("database error: %w", err)
}
