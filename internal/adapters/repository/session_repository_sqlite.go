package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/sqlite"
	"github.com/matt-riley/mjrwtf/internal/domain/session"
)

// SQLiteSessionRepository implements the session repository for SQLite
type SQLiteSessionRepository struct {
	sessionRepositoryBase
	queries *sqliterepo.Queries
}

// NewSQLiteSessionRepository creates a new SQLite session repository
func NewSQLiteSessionRepository(db *sql.DB) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{
		sessionRepositoryBase: sessionRepositoryBase{db: db},
		queries:               sqliterepo.New(db),
	}
}

// Create creates a new session
func (r *SQLiteSessionRepository) Create(ctx context.Context, s *session.Session) error {
	// Convert domain entity to database params
	var ipAddress *string
	if s.IPAddress != "" {
		ipAddress = &s.IPAddress
	}

	var userAgent *string
	if s.UserAgent != "" {
		userAgent = &s.UserAgent
	}

	_, err := r.queries.CreateSession(ctx, sqliterepo.CreateSessionParams{
		ID:             s.ID,
		UserID:         s.UserID,
		CreatedAt:      s.CreatedAt,
		ExpiresAt:      s.ExpiresAt,
		LastActivityAt: s.LastActivityAt,
		IpAddress:      ipAddress,
		UserAgent:      userAgent,
	})

	if err != nil {
		return mapSessionSQLError(err)
	}

	return nil
}

// FindByID retrieves a session by its ID
func (r *SQLiteSessionRepository) FindByID(ctx context.Context, sessionID string) (*session.Session, error) {
	result, err := r.queries.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, mapSessionSQLError(err)
	}

	return &session.Session{
		ID:             result.ID,
		UserID:         result.UserID,
		CreatedAt:      result.CreatedAt,
		ExpiresAt:      result.ExpiresAt,
		LastActivityAt: result.LastActivityAt,
		IPAddress:      stringPtrToString(result.IpAddress),
		UserAgent:      stringPtrToString(result.UserAgent),
	}, nil
}

// FindByUserID retrieves all sessions for a specific user
func (r *SQLiteSessionRepository) FindByUserID(ctx context.Context, userID string) ([]*session.Session, error) {
	results, err := r.queries.ListSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, mapSessionSQLError(err)
	}

	sessions := make([]*session.Session, len(results))
	for i, result := range results {
		sessions[i] = &session.Session{
			ID:             result.ID,
			UserID:         result.UserID,
			CreatedAt:      result.CreatedAt,
			ExpiresAt:      result.ExpiresAt,
			LastActivityAt: result.LastActivityAt,
			IPAddress:      stringPtrToString(result.IpAddress),
			UserAgent:      stringPtrToString(result.UserAgent),
		}
	}

	return sessions, nil
}

// UpdateActivity updates the last activity timestamp for a session
func (r *SQLiteSessionRepository) UpdateActivity(ctx context.Context, sessionID string, lastActivityAt time.Time) error {
	err := r.queries.UpdateSessionActivity(ctx, sqliterepo.UpdateSessionActivityParams{
		ID:             sessionID,
		LastActivityAt: lastActivityAt,
	})

	if err != nil {
		return mapSessionSQLError(err)
	}

	return nil
}

// Delete removes a session by its ID
func (r *SQLiteSessionRepository) Delete(ctx context.Context, sessionID string) error {
	err := r.queries.DeleteSession(ctx, sessionID)
	if err != nil {
		return mapSessionSQLError(err)
	}

	return nil
}

// DeleteByUserID removes all sessions for a specific user
func (r *SQLiteSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	err := r.queries.DeleteSessionsByUserID(ctx, userID)
	if err != nil {
		return mapSessionSQLError(err)
	}

	return nil
}

// DeleteExpired removes all expired sessions
func (r *SQLiteSessionRepository) DeleteExpired(ctx context.Context) (int, error) {
	count, err := r.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return 0, mapSessionSQLError(err)
	}

	return int(count), nil
}

// DeleteIdle removes all sessions that have been idle longer than the specified timeout
func (r *SQLiteSessionRepository) DeleteIdle(ctx context.Context, idleTimeout time.Duration) (int, error) {
	threshold := time.Now().Add(-idleTimeout)
	count, err := r.queries.DeleteIdleSessions(ctx, threshold)
	if err != nil {
		return 0, mapSessionSQLError(err)
	}

	return int(count), nil
}

// stringPtrToString converts a string pointer to a string (empty if nil)
func stringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
