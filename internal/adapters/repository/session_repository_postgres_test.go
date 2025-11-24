package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/matt-riley/mjrwtf/internal/domain/session"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"
)

func setupPostgresSessionTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Get database connection string from environment or use default test database
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/mjrwtf_test?sslmode=disable"
	}

	// Try to connect to PostgreSQL
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
		return nil, nil
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
		return nil, nil
	}

	// Create a unique schema for this test to avoid conflicts
	schemaName := fmt.Sprintf("test_%d", time.Now().UnixNano())

	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Set search path to use the test schema
	_, err = db.Exec(fmt.Sprintf("SET search_path TO %s", schemaName))
	if err != nil {
		t.Fatalf("failed to set search path: %v", err)
	}

	// Run migrations
	goose.SetBaseFS(migrations.PostgresMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}

	if err := goose.Up(db, migrations.PostgresDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		db.Close()
	}

	return db, cleanup
}

func createTestSessionForPostgres(t *testing.T, userID, ipAddress, userAgent string) *session.Session {
	t.Helper()

	s, err := session.NewSession(userID, ipAddress, userAgent)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	return s
}

func TestPostgresSessionRepository_Create(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return // Test was skipped
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("successfully create session", func(t *testing.T) {
		s := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")

		err := repo.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Verify session was created
		found, err := repo.FindByID(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.ID != s.ID {
			t.Errorf("Create() ID = %v, want %v", found.ID, s.ID)
		}
		if found.UserID != s.UserID {
			t.Errorf("Create() UserID = %v, want %v", found.UserID, s.UserID)
		}
	})

	t.Run("create session with nullable fields", func(t *testing.T) {
		s := createTestSessionForPostgres(t, "user2", "", "")

		err := repo.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("Create() with nullable fields error = %v", err)
		}

		// Verify session was created with empty strings for nullable fields
		found, err := repo.FindByID(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.IPAddress != "" {
			t.Errorf("Create() IPAddress = %v, want empty string", found.IPAddress)
		}
		if found.UserAgent != "" {
			t.Errorf("Create() UserAgent = %v, want empty string", found.UserAgent)
		}
	})

	t.Run("create duplicate session returns error", func(t *testing.T) {
		s := createTestSessionForPostgres(t, "user3", "192.168.1.2", "Mozilla/5.0")

		// First create should succeed
		err := repo.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		// Second create with same ID should fail
		err = repo.Create(context.Background(), s)
		if err == nil {
			t.Error("Create() with duplicate ID should return error")
		}
	})
}

func TestPostgresSessionRepository_FindByID(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("find existing session", func(t *testing.T) {
		s := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")
		err := repo.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := repo.FindByID(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.ID != s.ID {
			t.Errorf("FindByID() ID = %v, want %v", found.ID, s.ID)
		}
		if found.UserID != s.UserID {
			t.Errorf("FindByID() UserID = %v, want %v", found.UserID, s.UserID)
		}
		if found.IPAddress != s.IPAddress {
			t.Errorf("FindByID() IPAddress = %v, want %v", found.IPAddress, s.IPAddress)
		}
		if found.UserAgent != s.UserAgent {
			t.Errorf("FindByID() UserAgent = %v, want %v", found.UserAgent, s.UserAgent)
		}
	})

	t.Run("session not found returns error", func(t *testing.T) {
		// Generate a valid session ID format
		nonExistentID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		_, err := repo.FindByID(context.Background(), nonExistentID)
		if err != session.ErrSessionNotFound {
			t.Errorf("FindByID() error = %v, want %v", err, session.ErrSessionNotFound)
		}
	})
}

func TestPostgresSessionRepository_FindByUserID(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("list sessions for user with multiple sessions", func(t *testing.T) {
		// Create multiple sessions for same user at different times
		s1 := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")
		s1.LastActivityAt = time.Now().Add(-2 * time.Hour)
		repo.Create(context.Background(), s1)

		time.Sleep(10 * time.Millisecond) // Ensure different timestamps

		s2 := createTestSessionForPostgres(t, "user1", "192.168.1.2", "Chrome/90.0")
		s2.LastActivityAt = time.Now().Add(-1 * time.Hour)
		repo.Create(context.Background(), s2)

		time.Sleep(10 * time.Millisecond)

		s3 := createTestSessionForPostgres(t, "user1", "192.168.1.3", "Safari/14.0")
		s3.LastActivityAt = time.Now()
		repo.Create(context.Background(), s3)

		// Create session for different user
		s4 := createTestSessionForPostgres(t, "user2", "192.168.1.4", "Firefox/88.0")
		repo.Create(context.Background(), s4)

		// Find sessions for user1
		sessions, err := repo.FindByUserID(context.Background(), "user1")
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}

		if len(sessions) != 3 {
			t.Errorf("FindByUserID() returned %d sessions, want 3", len(sessions))
		}

		// Verify ordering by last_activity_at DESC
		if len(sessions) >= 2 {
			if sessions[0].LastActivityAt.Before(sessions[1].LastActivityAt) {
				t.Error("FindByUserID() sessions not ordered by last_activity_at DESC")
			}
		}

		// Verify all sessions belong to user1
		for _, s := range sessions {
			if s.UserID != "user1" {
				t.Errorf("FindByUserID() returned session with UserID = %v, want user1", s.UserID)
			}
		}
	})

	t.Run("list sessions for user with no sessions", func(t *testing.T) {
		sessions, err := repo.FindByUserID(context.Background(), "nonexistentuser")
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}

		if len(sessions) != 0 {
			t.Errorf("FindByUserID() returned %d sessions, want 0", len(sessions))
		}
	})
}

func TestPostgresSessionRepository_UpdateActivity(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("update activity for existing session", func(t *testing.T) {
		s := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")
		originalActivity := s.LastActivityAt

		err := repo.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Wait a bit to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		newActivityTime := time.Now()
		err = repo.UpdateActivity(context.Background(), s.ID, newActivityTime)
		if err != nil {
			t.Fatalf("UpdateActivity() error = %v", err)
		}

		// Verify activity was updated
		found, err := repo.FindByID(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		// Check that LastActivityAt was updated (should be after original)
		if !found.LastActivityAt.After(originalActivity) {
			t.Errorf("UpdateActivity() LastActivityAt = %v, should be after %v", found.LastActivityAt, originalActivity)
		}
	})

	t.Run("update activity for non-existent session succeeds", func(t *testing.T) {
		// PostgreSQL UPDATE with no matching rows succeeds without error
		nonExistentID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		err := repo.UpdateActivity(context.Background(), nonExistentID, time.Now())
		if err != nil {
			t.Errorf("UpdateActivity() non-existent session error = %v, want nil", err)
		}
	})
}

func TestPostgresSessionRepository_Delete(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("delete existing session", func(t *testing.T) {
		s := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")
		err := repo.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		err = repo.Delete(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify session was deleted
		_, err = repo.FindByID(context.Background(), s.ID)
		if err != session.ErrSessionNotFound {
			t.Errorf("After Delete(), FindByID() error = %v, want %v", err, session.ErrSessionNotFound)
		}
	})

	t.Run("delete non-existent session succeeds", func(t *testing.T) {
		nonExistentID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		err := repo.Delete(context.Background(), nonExistentID)
		if err != nil {
			t.Errorf("Delete() non-existent session error = %v, want nil", err)
		}
	})
}

func TestPostgresSessionRepository_DeleteByUserID(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("delete all sessions for user", func(t *testing.T) {
		// Create multiple sessions for user1
		s1 := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")
		repo.Create(context.Background(), s1)

		s2 := createTestSessionForPostgres(t, "user1", "192.168.1.2", "Chrome/90.0")
		repo.Create(context.Background(), s2)

		s3 := createTestSessionForPostgres(t, "user1", "192.168.1.3", "Safari/14.0")
		repo.Create(context.Background(), s3)

		// Create session for different user
		s4 := createTestSessionForPostgres(t, "user2", "192.168.1.4", "Firefox/88.0")
		repo.Create(context.Background(), s4)

		// Delete all sessions for user1
		err := repo.DeleteByUserID(context.Background(), "user1")
		if err != nil {
			t.Fatalf("DeleteByUserID() error = %v", err)
		}

		// Verify user1 sessions were deleted
		sessions, err := repo.FindByUserID(context.Background(), "user1")
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("After DeleteByUserID(), FindByUserID() returned %d sessions, want 0", len(sessions))
		}

		// Verify user2 session still exists
		sessions, err = repo.FindByUserID(context.Background(), "user2")
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("After DeleteByUserID(), user2 sessions = %d, want 1", len(sessions))
		}
	})

	t.Run("delete sessions for user with no sessions succeeds", func(t *testing.T) {
		err := repo.DeleteByUserID(context.Background(), "nonexistentuser")
		if err != nil {
			t.Errorf("DeleteByUserID() for non-existent user error = %v, want nil", err)
		}
	})
}

func TestPostgresSessionRepository_DeleteExpired(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("delete expired sessions", func(t *testing.T) {
		now := time.Now()

		// Create expired session
		s1 := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")
		s1.ExpiresAt = now.Add(-1 * time.Hour)
		repo.Create(context.Background(), s1)

		// Create another expired session
		s2 := createTestSessionForPostgres(t, "user2", "192.168.1.2", "Chrome/90.0")
		s2.ExpiresAt = now.Add(-2 * time.Hour)
		repo.Create(context.Background(), s2)

		// Create active session
		s3 := createTestSessionForPostgres(t, "user3", "192.168.1.3", "Safari/14.0")
		s3.ExpiresAt = now.Add(1 * time.Hour)
		repo.Create(context.Background(), s3)

		// Delete expired sessions
		count, err := repo.DeleteExpired(context.Background())
		if err != nil {
			t.Fatalf("DeleteExpired() error = %v", err)
		}

		if count != 2 {
			t.Errorf("DeleteExpired() deleted %d sessions, want 2", count)
		}

		// Verify expired sessions were deleted
		_, err = repo.FindByID(context.Background(), s1.ID)
		if err != session.ErrSessionNotFound {
			t.Errorf("After DeleteExpired(), expired session 1 error = %v, want %v", err, session.ErrSessionNotFound)
		}

		_, err = repo.FindByID(context.Background(), s2.ID)
		if err != session.ErrSessionNotFound {
			t.Errorf("After DeleteExpired(), expired session 2 error = %v, want %v", err, session.ErrSessionNotFound)
		}

		// Verify active session still exists
		_, err = repo.FindByID(context.Background(), s3.ID)
		if err != nil {
			t.Errorf("After DeleteExpired(), active session error = %v, want nil", err)
		}
	})

	t.Run("handle empty table", func(t *testing.T) {
		count, err := repo.DeleteExpired(context.Background())
		if err != nil {
			t.Fatalf("DeleteExpired() on empty table error = %v", err)
		}

		if count != 0 {
			t.Errorf("DeleteExpired() on empty table deleted %d sessions, want 0", count)
		}
	})
}

func TestPostgresSessionRepository_DeleteIdle(t *testing.T) {
	db, cleanup := setupPostgresSessionTestDB(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	repo := NewPostgresSessionRepository(db)

	t.Run("delete idle sessions based on timeout", func(t *testing.T) {
		now := time.Now()

		// Create idle session (last activity 2 hours ago)
		s1 := createTestSessionForPostgres(t, "user1", "192.168.1.1", "Mozilla/5.0")
		s1.LastActivityAt = now.Add(-2 * time.Hour)
		repo.Create(context.Background(), s1)

		// Create another idle session (last activity 3 hours ago)
		s2 := createTestSessionForPostgres(t, "user2", "192.168.1.2", "Chrome/90.0")
		s2.LastActivityAt = now.Add(-3 * time.Hour)
		repo.Create(context.Background(), s2)

		// Create recently active session (last activity 30 minutes ago)
		s3 := createTestSessionForPostgres(t, "user3", "192.168.1.3", "Safari/14.0")
		s3.LastActivityAt = now.Add(-30 * time.Minute)
		repo.Create(context.Background(), s3)

		// Delete sessions idle for more than 1 hour
		count, err := repo.DeleteIdle(context.Background(), 1*time.Hour)
		if err != nil {
			t.Fatalf("DeleteIdle() error = %v", err)
		}

		if count != 2 {
			t.Errorf("DeleteIdle() deleted %d sessions, want 2", count)
		}

		// Verify idle sessions were deleted
		_, err = repo.FindByID(context.Background(), s1.ID)
		if err != session.ErrSessionNotFound {
			t.Errorf("After DeleteIdle(), idle session 1 error = %v, want %v", err, session.ErrSessionNotFound)
		}

		_, err = repo.FindByID(context.Background(), s2.ID)
		if err != session.ErrSessionNotFound {
			t.Errorf("After DeleteIdle(), idle session 2 error = %v, want %v", err, session.ErrSessionNotFound)
		}

		// Verify recently active session still exists
		_, err = repo.FindByID(context.Background(), s3.ID)
		if err != nil {
			t.Errorf("After DeleteIdle(), active session error = %v, want nil", err)
		}
	})

	t.Run("return correct count", func(t *testing.T) {
		now := time.Now()

		// Create 5 idle sessions
		for i := 0; i < 5; i++ {
			s := createTestSessionForPostgres(t, "user"+string(rune(i)), "192.168.1."+string(rune(i)), "Mozilla/5.0")
			s.LastActivityAt = now.Add(-2 * time.Hour)
			repo.Create(context.Background(), s)
		}

		// Delete idle sessions
		count, err := repo.DeleteIdle(context.Background(), 1*time.Hour)
		if err != nil {
			t.Fatalf("DeleteIdle() error = %v", err)
		}

		if count != 5 {
			t.Errorf("DeleteIdle() deleted %d sessions, want 5", count)
		}
	})
}
