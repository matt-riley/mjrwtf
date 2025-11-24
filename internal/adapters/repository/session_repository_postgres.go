package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/postgres"
	"github.com/matt-riley/mjrwtf/internal/domain/session"
)

// PostgresSessionRepository implements the session repository for PostgreSQL
type PostgresSessionRepository struct {
	sessionRepositoryBase
	queries *postgresrepo.Queries
}

// NewPostgresSessionRepository creates a new PostgreSQL session repository
func NewPostgresSessionRepository(db *sql.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{
		sessionRepositoryBase: sessionRepositoryBase{db: db},
		queries:               postgresrepo.New(db),
	}
}

// Create creates a new session
func (r *PostgresSessionRepository) Create(ctx context.Context, s *session.Session) error {
	// Parse session ID string to UUID
	sessionUUID, err := uuid.Parse(s.ID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	// Convert domain entity to database params
	ipAddress := sql.NullString{
		String: s.IPAddress,
		Valid:  s.IPAddress != "",
	}

	userAgent := sql.NullString{
		String: s.UserAgent,
		Valid:  s.UserAgent != "",
	}

	_, err = r.queries.CreateSession(ctx, postgresrepo.CreateSessionParams{
		ID:             sessionUUID,
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
func (r *PostgresSessionRepository) FindByID(ctx context.Context, sessionID string) (*session.Session, error) {
	// Parse session ID string to UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %w", err)
	}

	result, err := r.queries.GetSessionByID(ctx, sessionUUID)
	if err != nil {
		return nil, mapSessionSQLError(err)
	}

	return &session.Session{
		ID:             result.ID.String(),
		UserID:         result.UserID,
		CreatedAt:      result.CreatedAt,
		ExpiresAt:      result.ExpiresAt,
		LastActivityAt: result.LastActivityAt,
		IPAddress:      nullStringToString(result.IpAddress),
		UserAgent:      nullStringToString(result.UserAgent),
	}, nil
}

// FindByUserID retrieves all sessions for a specific user
func (r *PostgresSessionRepository) FindByUserID(ctx context.Context, userID string) ([]*session.Session, error) {
	results, err := r.queries.ListSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, mapSessionSQLError(err)
	}

	sessions := make([]*session.Session, len(results))
	for i, result := range results {
		sessions[i] = &session.Session{
			ID:             result.ID.String(),
			UserID:         result.UserID,
			CreatedAt:      result.CreatedAt,
			ExpiresAt:      result.ExpiresAt,
			LastActivityAt: result.LastActivityAt,
			IPAddress:      nullStringToString(result.IpAddress),
			UserAgent:      nullStringToString(result.UserAgent),
		}
	}

	return sessions, nil
}

// UpdateActivity updates the last activity timestamp for a session
func (r *PostgresSessionRepository) UpdateActivity(ctx context.Context, sessionID string, lastActivityAt time.Time) error {
	// Parse session ID string to UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	err = r.queries.UpdateSessionActivity(ctx, postgresrepo.UpdateSessionActivityParams{
		ID:             sessionUUID,
		LastActivityAt: lastActivityAt,
	})

	if err != nil {
		return mapSessionSQLError(err)
	}

	return nil
}

// Delete removes a session by its ID
func (r *PostgresSessionRepository) Delete(ctx context.Context, sessionID string) error {
	// Parse session ID string to UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	err = r.queries.DeleteSession(ctx, sessionUUID)
	if err != nil {
		return mapSessionSQLError(err)
	}

	return nil
}

// DeleteByUserID removes all sessions for a specific user
func (r *PostgresSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	err := r.queries.DeleteSessionsByUserID(ctx, userID)
	if err != nil {
		return mapSessionSQLError(err)
	}

	return nil
}

// DeleteExpired removes all expired sessions
func (r *PostgresSessionRepository) DeleteExpired(ctx context.Context) (int, error) {
	count, err := r.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return 0, mapSessionSQLError(err)
	}

	return int(count), nil
}

// DeleteIdle removes all sessions that have been idle longer than the specified timeout
func (r *PostgresSessionRepository) DeleteIdle(ctx context.Context, idleTimeout time.Duration) (int, error) {
	threshold := time.Now().Add(-idleTimeout)
	count, err := r.queries.DeleteIdleSessions(ctx, threshold)
	if err != nil {
		return 0, mapSessionSQLError(err)
	}

	return int(count), nil
}

// nullStringToString converts a sql.NullString to a string (empty if invalid)
func nullStringToString(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}
