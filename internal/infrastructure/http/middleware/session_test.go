package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository"
	"github.com/matt-riley/mjrwtf/internal/domain/session"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"
)

// setupSessionTestDB creates an in-memory SQLite database for testing
func setupSessionTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Run migrations
	goose.SetBaseFS(migrations.SQLiteMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}

	if err := goose.Up(db, migrations.SQLiteDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

// createTestSession creates a test session
func createTestSession(t *testing.T, userID, ipAddress, userAgent string) *session.Session {
	t.Helper()

	s, err := session.NewSession(userID, ipAddress, userAgent)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	return s
}

func TestNewSessionMiddleware(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)

	t.Run("creates middleware with default timeout", func(t *testing.T) {
		sm := NewSessionMiddleware(repo, 0, false)

		if sm.sessionRepo == nil {
			t.Error("expected session repository to be set")
		}

		if sm.sessionTimeout != DefaultSessionTimeout {
			t.Errorf("expected default timeout %v, got %v", DefaultSessionTimeout, sm.sessionTimeout)
		}

		if sm.secureCookies {
			t.Error("expected secure cookies to be false")
		}
	})

	t.Run("creates middleware with custom timeout", func(t *testing.T) {
		customTimeout := 15 * time.Minute
		sm := NewSessionMiddleware(repo, customTimeout, true)

		if sm.sessionTimeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, sm.sessionTimeout)
		}

		if !sm.secureCookies {
			t.Error("expected secure cookies to be true")
		}
	})
}

func TestRequireSession_ValidSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	// Create and store a test session
	testSession := createTestSession(t, "test-user", "192.168.1.1", "Mozilla/5.0")
	if err := repo.Create(context.Background(), testSession); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	var capturedSession *session.Session
	var capturedUserID string
	handler := sm.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSession, _ = GetSession(r.Context())
		capturedUserID, _ = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if capturedSession == nil {
		t.Fatal("expected session in context")
	}

	if capturedSession.ID != testSession.ID {
		t.Errorf("expected session ID %s, got %s", testSession.ID, capturedSession.ID)
	}

	if capturedUserID != "test-user" {
		t.Errorf("expected user ID 'test-user', got %s", capturedUserID)
	}

	if rec.Body.String() != "success" {
		t.Errorf("expected body 'success', got %s", rec.Body.String())
	}
}

func TestRequireSession_MissingCookie(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	handler := sm.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"Unauthorized: missing session"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}
}

func TestRequireSession_InvalidSessionIDFormat(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	tests := []struct {
		name      string
		sessionID string
	}{
		{"empty session ID", ""},
		{"too short", "abc123"},
		{"invalid characters", "invalid-session-id-with-invalid-chars-12345678901234567890"},
		{"wrong length", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"}, // 63 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := sm.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called")
			}))

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.sessionID != "" {
				req.AddCookie(&http.Cookie{
					Name:  SessionCookieName,
					Value: tt.sessionID,
				})
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
			}
		})
	}
}

func TestRequireSession_SessionNotFound(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	handler := sm.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	// Use a valid format session ID that doesn't exist in the database
	nonExistentSessionID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: nonExistentSessionID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"Unauthorized: session not found"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}
}

func TestRequireSession_ExpiredSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	// Create a session that's already expired
	testSession := createTestSession(t, "test-user", "192.168.1.1", "Mozilla/5.0")
	testSession.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired 1 hour ago

	if err := repo.Create(context.Background(), testSession); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	handler := sm.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"Unauthorized: session expired"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}
}

func TestRequireSession_IdleSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	idleTimeout := 30 * time.Minute
	sm := NewSessionMiddleware(repo, idleTimeout, false)

	// Create a session with old last activity
	testSession := createTestSession(t, "test-user", "192.168.1.1", "Mozilla/5.0")
	testSession.LastActivityAt = time.Now().Add(-1 * time.Hour) // Last activity 1 hour ago

	if err := repo.Create(context.Background(), testSession); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	handler := sm.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"Unauthorized: session idle timeout"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}
}

func TestRequireSession_UpdatesActivityTimestamp(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	// Create a session with an old activity timestamp
	testSession := createTestSession(t, "test-user", "192.168.1.1", "Mozilla/5.0")
	oldActivity := time.Now().Add(-5 * time.Minute)
	testSession.LastActivityAt = oldActivity

	if err := repo.Create(context.Background(), testSession); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	handler := sm.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify activity was updated in database
	updatedSession, err := repo.FindByID(context.Background(), testSession.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated session: %v", err)
	}

	// Activity should be more recent than the old timestamp
	if !updatedSession.LastActivityAt.After(oldActivity) {
		t.Errorf("expected activity to be updated, got %v (original was %v)", updatedSession.LastActivityAt, oldActivity)
	}
}

func TestOptionalSession_NoCookie(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	var sessionFound bool
	handler := sm.OptionalSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, sessionFound = GetSession(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if sessionFound {
		t.Error("expected no session in context")
	}

	if rec.Body.String() != "success" {
		t.Errorf("expected body 'success', got %s", rec.Body.String())
	}
}

func TestOptionalSession_ValidSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	// Create and store a test session
	testSession := createTestSession(t, "test-user", "192.168.1.1", "Mozilla/5.0")
	if err := repo.Create(context.Background(), testSession); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	var capturedSession *session.Session
	var sessionFound bool
	handler := sm.OptionalSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSession, sessionFound = GetSession(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !sessionFound {
		t.Fatal("expected session in context")
	}

	if capturedSession.ID != testSession.ID {
		t.Errorf("expected session ID %s, got %s", testSession.ID, capturedSession.ID)
	}
}

func TestOptionalSession_InvalidSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	sm := NewSessionMiddleware(repo, 30*time.Minute, false)

	tests := []struct {
		name      string
		sessionID string
	}{
		{"invalid format", "invalid-session-id"},
		{"not found", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sessionFound bool
			handler := sm.OptionalSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, sessionFound = GetSession(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  SessionCookieName,
				Value: tt.sessionID,
			})
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			if sessionFound {
				t.Error("expected no session in context")
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	testSession := &session.Session{
		ID:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		UserID: "test-user",
	}

	t.Run("returns session from context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), SessionKey, testSession)

		sess, ok := GetSession(ctx)

		if !ok {
			t.Error("expected session to be found")
		}

		if sess.ID != testSession.ID {
			t.Errorf("expected session ID %s, got %s", testSession.ID, sess.ID)
		}
	})

	t.Run("returns false if no session", func(t *testing.T) {
		ctx := context.Background()

		_, ok := GetSession(ctx)

		if ok {
			t.Error("expected session not to be found")
		}
	})
}

func TestSetSessionCookie(t *testing.T) {
	sessionID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name          string
		maxAge        int
		secureCookies bool
	}{
		{"non-secure cookie", 3600, false},
		{"secure cookie", 3600, true},
		{"with max age", 86400, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			SetSessionCookie(rec, sessionID, tt.maxAge, tt.secureCookies)

			cookies := rec.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]

			if cookie.Name != SessionCookieName {
				t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
			}

			if cookie.Value != sessionID {
				t.Errorf("expected cookie value %s, got %s", sessionID, cookie.Value)
			}

			if cookie.Path != "/" {
				t.Errorf("expected path '/', got %s", cookie.Path)
			}

			if cookie.MaxAge != tt.maxAge {
				t.Errorf("expected max age %d, got %d", tt.maxAge, cookie.MaxAge)
			}

			if !cookie.HttpOnly {
				t.Error("expected HttpOnly flag to be true")
			}

			if cookie.Secure != tt.secureCookies {
				t.Errorf("expected Secure flag to be %v, got %v", tt.secureCookies, cookie.Secure)
			}

			if cookie.SameSite != http.SameSiteLaxMode {
				t.Errorf("expected SameSite to be Lax, got %v", cookie.SameSite)
			}
		})
	}
}

func TestClearSessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()

	ClearSessionCookie(rec)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]

	if cookie.Name != SessionCookieName {
		t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
	}

	if cookie.Value != "" {
		t.Errorf("expected empty cookie value, got %s", cookie.Value)
	}

	if cookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", cookie.MaxAge)
	}

	if !cookie.HttpOnly {
		t.Error("expected HttpOnly flag to be true")
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite to be Lax, got %v", cookie.SameSite)
	}
}
