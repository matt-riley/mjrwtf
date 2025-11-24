package session

import (
	"context"
	"time"
)

// Repository defines the interface for Session persistence operations
// Following hexagonal architecture, this interface is defined in the domain layer
// and implemented by adapters (e.g., PostgreSQL, SQLite)
type Repository interface {
	// Create creates a new session
	Create(ctx context.Context, session *Session) error

	// FindByID retrieves a session by its ID
	// Returns ErrSessionNotFound if the session doesn't exist
	FindByID(ctx context.Context, sessionID string) (*Session, error)

	// FindByUserID retrieves all sessions for a specific user
	FindByUserID(ctx context.Context, userID string) ([]*Session, error)

	// UpdateActivity updates the last activity timestamp for a session
	// Returns ErrSessionNotFound if the session doesn't exist
	UpdateActivity(ctx context.Context, sessionID string, lastActivityAt time.Time) error

	// Delete removes a session by its ID
	// Returns ErrSessionNotFound if the session doesn't exist
	Delete(ctx context.Context, sessionID string) error

	// DeleteByUserID removes all sessions for a specific user
	DeleteByUserID(ctx context.Context, userID string) error

	// DeleteExpired removes all expired sessions
	// Returns the number of sessions deleted
	DeleteExpired(ctx context.Context) (int, error)

	// DeleteIdle removes all sessions that have been idle longer than the specified timeout
	// Returns the number of sessions deleted
	DeleteIdle(ctx context.Context, idleTimeout time.Duration) (int, error)
}
