package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
)

// TestServer_RateLimitRedirect tests rate limiting on the public redirect endpoint
func TestServer_RateLimitRedirect(t *testing.T) {
	cfg := &config.Config{
		ServerPort:             8080,
		BaseURL:                "http://localhost:8080",
		DatabaseURL:            "test.db",
		AuthToken:              "test-token",
		AllowedOrigins:         "*",
		RateLimitEnabled:       true,
		RateLimitRedirectRPS:   1,
		RateLimitRedirectBurst: 2,
		RateLimitAPIRPS:        5,
		RateLimitAPIBurst:      10,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Create test URL
	testURL := &url.URL{
		ShortCode:   "testrl",
		OriginalURL: "https://example.com/test",
		CreatedBy:   "test-user",
		CreatedAt:   time.Now(),
	}
	urlRepo := repository.NewSQLiteURLRepository(db)
	ctx := context.Background()
	if err := urlRepo.Create(ctx, testURL); err != nil {
		t.Fatalf("failed to create test URL: %v", err)
	}

	// First two requests should succeed (within burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/testrl", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		srv.Router().ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusFound, rec.Code)
		}
	}

	// Third immediate request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/testrl", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("third request: expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Check for Retry-After header
	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header should be present")
	}
}

// TestServer_RateLimitRedirect_DifferentIPs tests that different IPs get separate rate limits
func TestServer_RateLimitRedirect_DifferentIPs(t *testing.T) {
	cfg := &config.Config{
		ServerPort:             8080,
		BaseURL:                "http://localhost:8080",
		DatabaseURL:            "test.db",
		AuthToken:              "test-token",
		AllowedOrigins:         "*",
		RateLimitEnabled:       true,
		RateLimitRedirectRPS:   1,
		RateLimitRedirectBurst: 1,
		RateLimitAPIRPS:        5,
		RateLimitAPIBurst:      10,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Create test URL
	testURL := &url.URL{
		ShortCode:   "testrl2",
		OriginalURL: "https://example.com/test",
		CreatedBy:   "test-user",
		CreatedAt:   time.Now(),
	}
	urlRepo := repository.NewSQLiteURLRepository(db)
	ctx := context.Background()
	if err := urlRepo.Create(ctx, testURL); err != nil {
		t.Fatalf("failed to create test URL: %v", err)
	}

	// Request from IP 1
	req1 := httptest.NewRequest(http.MethodGet, "/testrl2", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusFound {
		t.Errorf("IP1 request: expected status %d, got %d", http.StatusFound, rec1.Code)
	}

	// Immediate request from IP 2 should succeed (different bucket)
	req2 := httptest.NewRequest(http.MethodGet, "/testrl2", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusFound {
		t.Errorf("IP2 request: expected status %d, got %d", http.StatusFound, rec2.Code)
	}

	// Second request from IP 1 should be rate limited
	req3 := httptest.NewRequest(http.MethodGet, "/testrl2", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	rec3 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 second request: expected status %d, got %d", http.StatusTooManyRequests, rec3.Code)
	}
}

// TestServer_RateLimitRedirect_Disabled tests that rate limiting can be disabled
func TestServer_RateLimitRedirect_Disabled(t *testing.T) {
	cfg := &config.Config{
		ServerPort:             8080,
		BaseURL:                "http://localhost:8080",
		DatabaseURL:            "test.db",
		AuthToken:              "test-token",
		AllowedOrigins:         "*",
		RateLimitEnabled:       false, // Disabled
		RateLimitRedirectRPS:   1,
		RateLimitRedirectBurst: 1,
		RateLimitAPIRPS:        5,
		RateLimitAPIBurst:      10,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Create test URL
	testURL := &url.URL{
		ShortCode:   "testrl3",
		OriginalURL: "https://example.com/test",
		CreatedBy:   "test-user",
		CreatedAt:   time.Now(),
	}
	urlRepo := repository.NewSQLiteURLRepository(db)
	ctx := context.Background()
	if err := urlRepo.Create(ctx, testURL); err != nil {
		t.Fatalf("failed to create test URL: %v", err)
	}

	// Multiple rapid requests should all succeed when rate limiting is disabled
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/testrl3", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		srv.Router().ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("request %d: expected status %d, got %d (rate limiting should be disabled)", i+1, http.StatusFound, rec.Code)
		}
		
		// Small delay to allow async click recording to complete
		time.Sleep(10 * time.Millisecond)
	}
}

// TestServer_RateLimitAPI tests rate limiting on authenticated API endpoints
func TestServer_RateLimitAPI(t *testing.T) {
	cfg := &config.Config{
		ServerPort:             8080,
		BaseURL:                "http://localhost:8080",
		DatabaseURL:            "test.db",
		AuthToken:              "test-token",
		AllowedOrigins:         "*",
		RateLimitEnabled:       true,
		RateLimitRedirectRPS:   10,
		RateLimitRedirectBurst: 20,
		RateLimitAPIRPS:        2,
		RateLimitAPIBurst:      3,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// First three requests should succeed (within burst)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		srv.Router().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	// Fourth immediate request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("fourth request: expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Check for Retry-After header
	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header should be present")
	}
}

// TestServer_RateLimitAPI_GlobalLimit tests that API rate limit is global (not per-IP)
func TestServer_RateLimitAPI_GlobalLimit(t *testing.T) {
	cfg := &config.Config{
		ServerPort:             8080,
		BaseURL:                "http://localhost:8080",
		DatabaseURL:            "test.db",
		AuthToken:              "test-token",
		AllowedOrigins:         "*",
		RateLimitEnabled:       true,
		RateLimitRedirectRPS:   10,
		RateLimitRedirectBurst: 20,
		RateLimitAPIRPS:        2,
		RateLimitAPIBurst:      2,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Request from IP 1
	req1 := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req1.Header.Set("Authorization", "Bearer test-token")
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("IP1 request 1: expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Request from IP 2
	req2 := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req2.Header.Set("Authorization", "Bearer test-token")
	req2.RemoteAddr = "192.168.1.2:12345"
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("IP2 request 1: expected status %d, got %d", http.StatusOK, rec2.Code)
	}

	// Third request from IP 3 should be rate limited (global limit exceeded)
	req3 := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req3.Header.Set("Authorization", "Bearer test-token")
	req3.RemoteAddr = "192.168.1.3:12345"
	rec3 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("IP3 request: expected status %d, got %d (global rate limit should apply)", http.StatusTooManyRequests, rec3.Code)
	}
}

// TestServer_RateLimitAPI_Disabled tests that API rate limiting can be disabled
func TestServer_RateLimitAPI_Disabled(t *testing.T) {
	cfg := &config.Config{
		ServerPort:             8080,
		BaseURL:                "http://localhost:8080",
		DatabaseURL:            "test.db",
		AuthToken:              "test-token",
		AllowedOrigins:         "*",
		RateLimitEnabled:       false, // Disabled
		RateLimitRedirectRPS:   10,
		RateLimitRedirectBurst: 20,
		RateLimitAPIRPS:        1,
		RateLimitAPIBurst:      1,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Multiple rapid requests should all succeed when rate limiting is disabled
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		srv.Router().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d (rate limiting should be disabled)", i+1, http.StatusOK, rec.Code)
		}
	}
}
