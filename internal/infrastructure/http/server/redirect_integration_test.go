package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
)

// TestServer_RedirectEndpoint tests the public redirect endpoint
func TestServer_RedirectEndpoint(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Create a test URL directly via repository
	urlRepo := repository.NewSQLiteURLRepository(db)
	testURL := &url.URL{
		ShortCode:   "test123",
		OriginalURL: "https://example.com/test",
		CreatedBy:   "test-user",
		CreatedAt:   time.Now(),
	}

	ctx := context.Background()
	if err := urlRepo.Create(ctx, testURL); err != nil {
		t.Fatalf("failed to create test URL: %v", err)
	}

	tests := []struct {
		name               string
		shortCode          string
		referrer           string
		userAgent          string
		expectedStatus     int
		expectedLocation   string
		checkLocation      bool
	}{
		{
			name:             "successful redirect",
			shortCode:        "test123",
			referrer:         "https://google.com",
			userAgent:        "Mozilla/5.0",
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com/test",
			checkLocation:    true,
		},
		{
			name:             "redirect without referrer",
			shortCode:        "test123",
			referrer:         "",
			userAgent:        "Mozilla/5.0",
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com/test",
			checkLocation:    true,
		},
		{
			name:           "short code not found",
			shortCode:      "notfound",
			expectedStatus: http.StatusNotFound,
			checkLocation:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortCode, nil)
			if tt.referrer != "" {
				req.Header.Set("Referer", tt.referrer)
			}
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkLocation {
				location := rec.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected location '%s', got '%s'", tt.expectedLocation, location)
				}
			}
		})
	}
}

// TestServer_RedirectWithAnalytics tests that analytics are tracked
func TestServer_RedirectWithAnalytics(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Create a test URL directly via repository
	urlRepo := repository.NewSQLiteURLRepository(db)
	clickRepo := repository.NewSQLiteClickRepository(db)
	
	testURL := &url.URL{
		ShortCode:   "analytics123",
		OriginalURL: "https://example.com/analytics",
		CreatedBy:   "test-user",
		CreatedAt:   time.Now(),
	}

	ctx := context.Background()
	if err := urlRepo.Create(ctx, testURL); err != nil {
		t.Fatalf("failed to create test URL: %v", err)
	}

	// Make redirect request
	req := httptest.NewRequest(http.MethodGet, "/analytics123", nil)
	req.Header.Set("Referer", "https://google.com")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	// Wait a moment for async analytics to be recorded
	time.Sleep(100 * time.Millisecond)

	// Verify click was recorded
	// Get the URL from database to get its ID
	savedURL, err := urlRepo.FindByShortCode(ctx, "analytics123")
	if err != nil {
		t.Fatalf("failed to find URL: %v", err)
	}

	clickCount, err := clickRepo.GetTotalClickCount(ctx, savedURL.ID)
	if err != nil {
		t.Fatalf("failed to get click count: %v", err)
	}

	if clickCount != 1 {
		t.Errorf("expected 1 click, got %d", clickCount)
	}
}

// TestServer_RedirectPreservesURL tests that redirects preserve the original URL intact
func TestServer_RedirectPreservesURL(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Create test URLs with complex query parameters
	urlRepo := repository.NewSQLiteURLRepository(db)
	tests := []struct {
		name        string
		shortCode   string
		originalURL string
	}{
		{
			name:        "URL with query parameters",
			shortCode:   "query123",
			originalURL: "https://example.com/path?query=value&foo=bar",
		},
		{
			name:        "URL with encoded characters",
			shortCode:   "encoded123",
			originalURL: "https://example.com/path?q=hello%20world&special=%26%3D%3F",
		},
		{
			name:        "URL with fragment",
			shortCode:   "fragment123",
			originalURL: "https://example.com/page#section",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		testURL := &url.URL{
			ShortCode:   tt.shortCode,
			OriginalURL: tt.originalURL,
			CreatedBy:   "test-user",
			CreatedAt:   time.Now(),
		}
		if err := urlRepo.Create(ctx, testURL); err != nil {
			t.Fatalf("failed to create test URL: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortCode, nil)
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
			}

			location := rec.Header().Get("Location")
			if location != tt.originalURL {
				t.Errorf("expected location '%s', got '%s'", tt.originalURL, location)
			}
		})
	}
}

// TestServer_RedirectVsAPIRoutes tests that redirect routes don't conflict with API routes
func TestServer_RedirectVsAPIRoutes(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "health endpoint works",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API endpoint requires auth",
			path:           "/api/urls",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "nonexistent short code returns 404",
			path:           "/notfound",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := http.MethodGet
			if strings.Contains(tt.path, "/api/urls") && tt.path == "/api/urls" {
				// For API endpoints, use GET to list
				method = http.MethodGet
			}
			
			req := httptest.NewRequest(method, tt.path, nil)
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
