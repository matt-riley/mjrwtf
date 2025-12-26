package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/rs/zerolog"
)

// testLogger returns a disabled logger for tests
func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

func TestNew_CreatesServerWithMiddleware(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if srv == nil {
		t.Fatal("expected server to be created")
	}

	if srv.router == nil {
		t.Error("expected router to be initialized")
	}

	if srv.config != cfg {
		t.Error("expected config to be set")
	}
}

func TestHealthCheckHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expected := `{"status":"ok"}`
	if rec.Body.String() != expected {
		t.Errorf("expected response %s, got %s", expected, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestServer_MiddlewareOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test that recovery middleware works (middleware is active)
	srv.router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	// Recovery middleware should catch the panic and return 500
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d from recovery middleware, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestServer_CORSMiddleware(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	// CORS middleware should add appropriate headers
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected CORS headers to be set")
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		ServerPort:     0, // Use random available port
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start server in background with error handling
	serverErrors := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Give server time to start listening
	// Note: With port 0, we can't dial to check readiness, so we use a small delay
	time.Sleep(100 * time.Millisecond)

	// Check for startup errors
	select {
	case err := <-serverErrors:
		t.Fatalf("server failed to start: %v", err)
	default:
		// Server started successfully
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}
}

func TestServer_Timeouts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Verify timeout configurations
	if srv.httpServer.ReadTimeout != readTimeout {
		t.Errorf("expected ReadTimeout %v, got %v", readTimeout, srv.httpServer.ReadTimeout)
	}

	if srv.httpServer.WriteTimeout != writeTimeout {
		t.Errorf("expected WriteTimeout %v, got %v", writeTimeout, srv.httpServer.WriteTimeout)
	}

	if srv.httpServer.IdleTimeout != idleTimeout {
		t.Errorf("expected IdleTimeout %v, got %v", idleTimeout, srv.httpServer.IdleTimeout)
	}
}

func TestServer_ListenAddress(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
	}{
		{"default port", 8080, ":8080"},
		{"custom port", 3000, ":3000"},
		{"high port", 9999, ":9999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			cfg := &config.Config{
				ServerPort:     tt.port,
				BaseURL:        "http://localhost:8080",
				DatabaseURL:    "test.db",
				AuthToken:      "test-token",
				AllowedOrigins: "*",
			}

			srv, err := New(cfg, db, testLogger())
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}

			if srv.httpServer.Addr != tt.want {
				t.Errorf("expected address %s, got %s", tt.want, srv.httpServer.Addr)
			}
		})
	}
}

func TestServer_Router(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	router := srv.Router()
	if router == nil {
		t.Error("expected router to be returned")
	}

	if router != srv.router {
		t.Error("expected same router instance to be returned")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// First, make some requests to generate metrics
	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRec := httptest.NewRecorder()
	srv.router.ServeHTTP(healthRec, healthReq)

	// Now test the /metrics endpoint
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()

	// Verify expected Prometheus metrics are present
	// Note: Counter vectors may not appear until they have values
	expectedMetrics := []string{
		"mjrwtf_http_requests_total",
		"mjrwtf_http_request_duration_seconds",
		"mjrwtf_urls_active_total",
		"go_goroutines",
		"go_gc_duration_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metric %s in output", metric)
		}
	}
}

func TestServer_Metrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	m := srv.Metrics()
	if m == nil {
		t.Error("expected metrics to be returned")
	}

	if m.Registry == nil {
		t.Error("expected metrics registry to be initialized")
	}
}

func TestMetricsEndpoint_WithAuthEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		ServerPort:         8080,
		BaseURL:            "http://localhost:8080",
		DatabaseURL:        "test.db",
		AuthToken:          "test-token",
		AllowedOrigins:     "*",
		MetricsAuthEnabled: true,
	}

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "without auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "with invalid auth token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "with valid auth token",
			authHeader:     "Bearer test-token",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// For successful requests, verify metrics are present
			if tt.expectedStatus == http.StatusOK {
				body := rec.Body.String()
				// Check for Go runtime metrics which should always be present
				if !strings.Contains(body, "go_goroutines") {
					t.Error("expected go_goroutines metric in response body")
				}
			}
		})
	}
}

func TestMetricsEndpoint_WithAuthDisabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		ServerPort:         8080,
		BaseURL:            "http://localhost:8080",
		DatabaseURL:        "test.db",
		AuthToken:          "test-token",
		AllowedOrigins:     "*",
		MetricsAuthEnabled: false,
	}

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// When auth is disabled, /metrics should be accessible without authentication
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	// Check for Go runtime metrics which should always be present
	if !strings.Contains(body, "go_goroutines") {
		t.Error("expected go_goroutines metric in response body")
	}
}

func ExampleServer_Start() {
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "./database.db",
		AuthToken:      "secret-token",
		AllowedOrigins: "*",
	}

	db, _ := sql.Open("sqlite3", cfg.DatabaseURL)
	defer db.Close()

	logger := zerolog.Nop()
	srv, err := New(cfg, db, logger)
	if err != nil {
		fmt.Printf("Server creation error: %v\n", err)
		return
	}

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Graceful shutdown on signal...
	ctx := context.Background()
	srv.Shutdown(ctx)
}
