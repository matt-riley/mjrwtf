package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/session"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"

	_ "github.com/mattn/go-sqlite3"
)

// TestSessionBasedAPIAuthentication tests that dashboard users with valid sessions
// can successfully call API endpoints using session cookies instead of Bearer tokens
func TestSessionBasedAPIAuthentication(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Run migrations using embedded migrations
	goose.SetBaseFS(migrations.SQLiteMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}
	if err := goose.Up(db, "sqlite"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		DatabaseURL:   ":memory:",
		ServerPort:    8080,
		BaseURL:       "http://localhost:8080",
		AllowedOrigins: "*",
		AuthToken:     "test-token",
		SecureCookies: false,
		LogLevel:      "error",
		LogFormat:     "json",
	}

	// Create server
	logger := zerolog.Nop()
	server, err := New(cfg, db, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Shutdown(context.Background())

	// Test 1: Create URL with Bearer token (baseline)
	t.Run("create URL with Bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(`{"original_url":"https://example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Test 2: Create a session
	sessionStore := server.sessionStore
	sess, err := sessionStore.Create("authenticated-user")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Test 3: List URLs with session cookie
	t.Run("list URLs with session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: sess.ID,
		})

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Test 4: Delete URL with session cookie
	t.Run("delete URL with session cookie", func(t *testing.T) {
		// First, create a URL to delete using Bearer token
		createReq := httptest.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(`{"original_url":"https://delete-me.com"}`))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer test-token")
		
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		if createW.Code != http.StatusCreated {
			t.Fatalf("failed to create URL: %d: %s", createW.Code, createW.Body.String())
		}

		// Extract short code from response
		body := createW.Body.String()
		shortCodeStart := strings.Index(body, `"short_code":"`) + len(`"short_code":"`)
		shortCodeEnd := strings.Index(body[shortCodeStart:], `"`)
		shortCode := body[shortCodeStart : shortCodeStart+shortCodeEnd]

		// Now delete it using session cookie
		deleteReq := httptest.NewRequest(http.MethodDelete, "/api/urls/"+shortCode, nil)
		deleteReq.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: sess.ID,
		})

		deleteW := httptest.NewRecorder()
		server.router.ServeHTTP(deleteW, deleteReq)

		if deleteW.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", deleteW.Code, deleteW.Body.String())
		}
	})

	// Test 5: Verify expired session is rejected
	t.Run("expired session is rejected", func(t *testing.T) {
		// Create a session with very short TTL
		shortTTLStore := session.NewStore(1 * time.Millisecond)
		defer shortTTLStore.Shutdown()
		
		expiredSess, err := shortTTLStore.Create("test-user")
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		// Wait for it to expire
		time.Sleep(10 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: expiredSess.ID,
		})

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should fall back to Bearer token auth and fail (no token provided)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	// Test 6: Verify invalid session ID is rejected
	t.Run("invalid session ID is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: "invalid-session-id",
		})

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should fall back to Bearer token auth and fail (no token provided)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

// TestSessionAndBearerAuthCoexist tests that both authentication methods work
// and can be used interchangeably
func TestSessionAndBearerAuthCoexist(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Run migrations using embedded migrations
	goose.SetBaseFS(migrations.SQLiteMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}
	if err := goose.Up(db, "sqlite"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		DatabaseURL:   ":memory:",
		ServerPort:    8080,
		BaseURL:       "http://localhost:8080",
		AllowedOrigins: "*",
		AuthToken:     "test-token",
		SecureCookies: false,
		LogLevel:      "error",
		LogFormat:     "json",
	}

	// Create server
	logger := zerolog.Nop()
	server, err := New(cfg, db, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Shutdown(context.Background())

	// Create a session
	sess, err := server.sessionStore.Create("authenticated-user")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Test: Create URL with session, list with Bearer token
	t.Run("create with session, list with Bearer token", func(t *testing.T) {
		// Create with session
		createReq := httptest.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(`{"original_url":"https://mixed-auth.com"}`))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: sess.ID,
		})

		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		if createW.Code != http.StatusCreated {
			t.Errorf("create failed: %d: %s", createW.Code, createW.Body.String())
		}

		// List with Bearer token
		listReq := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		listReq.Header.Set("Authorization", "Bearer test-token")

		listW := httptest.NewRecorder()
		server.router.ServeHTTP(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Errorf("list failed: %d: %s", listW.Code, listW.Body.String())
		}
	})
}
