package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository"
	"github.com/matt-riley/mjrwtf/internal/domain/session"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"
)

// setupAuthTestDB creates an in-memory SQLite database for testing
func setupAuthTestDB(t *testing.T) (*sql.DB, func()) {
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

// createAndStoreSession creates a test session and stores it in the repository
func createAndStoreSession(t *testing.T, repo session.Repository, userID, ipAddress, userAgent string) *session.Session {
	t.Helper()

	s, err := session.NewSession(userID, ipAddress, userAgent)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("failed to store test session: %v", err)
	}

	return s
}

// withSession adds a session to the request context
func withSession(ctx context.Context, sess *session.Session) context.Context {
	return context.WithValue(ctx, middleware.SessionKey, sess)
}

func TestNewAuthHandler(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	authToken := "test-token"
	sessionTimeout := 30 * time.Minute

	handler := NewAuthHandler(repo, authToken, sessionTimeout, false)

	if handler.sessionRepo == nil {
		t.Error("expected session repository to be set")
	}

	if handler.authToken != authToken {
		t.Errorf("expected auth token %s, got %s", authToken, handler.authToken)
	}

	if handler.sessionTimeout != sessionTimeout {
		t.Errorf("expected session timeout %v, got %v", sessionTimeout, handler.sessionTimeout)
	}

	if handler.secureCookies {
		t.Error("expected secure cookies to be false")
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	authToken := "test-secret-token"
	handler := NewAuthHandler(repo, authToken, 30*time.Minute, false)

	requestBody := `{"token":"test-secret-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.RemoteAddr = "192.168.1.1:54321"

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Message != "Login successful" {
		t.Errorf("expected message 'Login successful', got %s", response.Message)
	}

	// Verify session cookie was set
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != middleware.SessionCookieName {
		t.Errorf("expected cookie name %s, got %s", middleware.SessionCookieName, cookie.Name)
	}

	if cookie.Value == "" {
		t.Error("expected session cookie value to be set")
	}

	// Verify session was created in database
	sess, err := repo.FindByID(context.Background(), cookie.Value)
	if err != nil {
		t.Fatalf("failed to find session: %v", err)
	}

	if sess.UserID != "authenticated-user" {
		t.Errorf("expected user ID 'authenticated-user', got %s", sess.UserID)
	}

	if sess.IPAddress != "192.168.1.1" {
		t.Errorf("expected IP address '192.168.1.1', got %s", sess.IPAddress)
	}

	if sess.UserAgent != "TestAgent/1.0" {
		t.Errorf("expected user agent 'TestAgent/1.0', got %s", sess.UserAgent)
	}
}

func TestAuthHandler_Login_MissingToken(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	requestBody := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	expected := `{"error":"token is required"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	requestBody := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	expected := `{"error":"invalid request body"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}
}

func TestAuthHandler_Login_WrongToken(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "correct-token", 30*time.Minute, false)

	requestBody := `{"token":"wrong-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"invalid token"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}

	// Verify no session was created
	cookies := rec.Result().Cookies()
	if len(cookies) != 0 {
		t.Errorf("expected no cookies, got %d", len(cookies))
	}
}

func TestAuthHandler_Login_IPAddressExtraction(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	tests := []struct {
		name           string
		remoteAddr     string
		expectedIP     string
	}{
		{"IPv4 with port", "192.168.1.1:54321", "192.168.1.1"},
		{"IPv4 without port", "192.168.1.1", "192.168.1.1"},
		{"IPv6 with brackets and port", "[2001:db8::1]:54321", "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := `{"token":"test-token"}`
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = tt.remoteAddr

			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			// Get session cookie
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			// Verify IP address in database
			sess, err := repo.FindByID(context.Background(), cookies[0].Value)
			if err != nil {
				t.Fatalf("failed to find session: %v", err)
			}

			if sess.IPAddress != tt.expectedIP {
				t.Errorf("expected IP address %s, got %s", tt.expectedIP, sess.IPAddress)
			}
		})
	}
}

func TestAuthHandler_Login_UserAgentExtraction(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	tests := []struct {
		name      string
		userAgent string
	}{
		{"Chrome browser", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"},
		{"Firefox browser", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0"},
		{"Custom agent", "CustomBot/1.0"},
		{"Empty agent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := `{"token":"test-token"}`
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			// Get session cookie
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			// Verify user agent in database
			sess, err := repo.FindByID(context.Background(), cookies[0].Value)
			if err != nil {
				t.Fatalf("failed to find session: %v", err)
			}

			if sess.UserAgent != tt.userAgent {
				t.Errorf("expected user agent %s, got %s", tt.userAgent, sess.UserAgent)
			}
		})
	}
}

func TestAuthHandler_Login_SecureCookies(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)

	tests := []struct {
		name          string
		secureCookies bool
	}{
		{"secure cookies enabled", true},
		{"secure cookies disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(repo, "test-token", 30*time.Minute, tt.secureCookies)

			requestBody := `{"token":"test-token"}`
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			cookies := rec.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			if cookies[0].Secure != tt.secureCookies {
				t.Errorf("expected Secure flag to be %v, got %v", tt.secureCookies, cookies[0].Secure)
			}
		})
	}
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	// Create a session
	testSession := createAndStoreSession(t, repo, "test-user", "192.168.1.1", "TestAgent/1.0")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req = req.WithContext(withSession(req.Context(), testSession))

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response LogoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Message != "Logout successful" {
		t.Errorf("expected message 'Logout successful', got %s", response.Message)
	}

	// Verify session cookie was cleared
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", cookie.MaxAge)
	}

	// Verify session was deleted from database
	_, err := repo.FindByID(context.Background(), testSession.ID)
	if err != session.ErrSessionNotFound {
		t.Errorf("expected session to be deleted, got error: %v", err)
	}
}

func TestAuthHandler_Logout_NoSession(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"unauthorized"}`
	body := rec.Body.String()
	body = body[:len(body)-1] // Remove trailing newline
	if body != expected {
		t.Errorf("expected body %s, got %s", expected, body)
	}
}

func TestAuthHandler_Logout_AlreadyDeletedSession(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	// Create a session but don't store it in the database
	// This simulates a session that was already deleted or expired
	testSession := &session.Session{
		ID:             "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		UserID:         "test-user",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		LastActivityAt: time.Now(),
		IPAddress:      "192.168.1.1",
		UserAgent:      "TestAgent/1.0",
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req = req.WithContext(withSession(req.Context(), testSession))

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	// Delete operation should succeed even if session doesn't exist
	// This is idempotent behavior - deleting a non-existent session is not an error
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response LogoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Message != "Logout successful" {
		t.Errorf("expected message 'Logout successful', got %s", response.Message)
	}
}

func TestAuthHandler_Login_MultipleConcurrentLogins(t *testing.T) {
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()

	repo := repository.NewSQLiteSessionRepository(db)
	handler := NewAuthHandler(repo, "test-token", 30*time.Minute, false)

	// Simulate multiple login requests
	numRequests := 5
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			requestBody := `{"token":"test-token"}`
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(requestBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Login(rec, req)
			results <- rec.Code
		}()
	}

	// Verify all requests succeeded
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, code)
		}
	}

	// Verify multiple sessions were created
	sessions, err := repo.FindByUserID(context.Background(), "authenticated-user")
	if err != nil {
		t.Fatalf("failed to find sessions: %v", err)
	}

	if len(sessions) != numRequests {
		t.Errorf("expected %d sessions, got %d", numRequests, len(sessions))
	}
}
