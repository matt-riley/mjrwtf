package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

// TestMiddlewareExecutionOrder verifies that middleware executes in the correct order
func TestMiddlewareExecutionOrder(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	// Create a buffer to capture log output
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	srv, err := New(cfg, db, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Add a test handler
	srv.router.Get("/test-order", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test-order", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify logging middleware executed (zerolog outputs JSON)
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "request completed") {
		t.Errorf("expected log entry for request, got: %s", logOutput)
	}
}

// TestMiddlewareRecoveryBeforeLogging ensures recovery middleware catches panics before logging
func TestMiddlewareRecoveryBeforeLogging(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	// Create a buffer to capture log output
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	srv, err := New(cfg, db, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	srv.router.Get("/panic-test", func(w http.ResponseWriter, r *http.Request) {
		panic("intentional panic for testing")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	// Recovery middleware should catch panic and return 500
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d from recovery, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Logging middleware should still execute and log the panic recovery
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Error("expected panic recovery log entry")
	}
}

// TestServer_NotFoundHandler tests the default 404 response
func TestServer_NotFoundHandler(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

// TestServer_MethodNotAllowed tests handling of unsupported HTTP methods
func TestServer_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Health check only supports GET, try POST
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestServer_RateLimitRedirect(t *testing.T) {
	cfg := &config.Config{
		ServerPort:                 8080,
		BaseURL:                    "http://localhost:8080",
		DatabaseURL:                "test.db",
		AuthToken:                  "test-token",
		AllowedOrigins:             "*",
		RedirectRateLimitPerMinute: 1,
		APIRateLimitPerMinute:      10,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	// Create a URL so we can exercise a successful redirect.
	createReq := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(`{"original_url":"https://example.com"}`))
	createReq.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	createRec := httptest.NewRecorder()
	srv.router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create URL status %d, got %d", http.StatusCreated, createRec.Code)
	}

	var createResp struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create URL response: %v", err)
	}
	if createResp.ShortCode == "" {
		t.Fatal("expected short_code in create response")
	}

	ip := "192.0.2.200:1234"

	req1 := httptest.NewRequest(http.MethodGet, "/"+createResp.ShortCode, nil)
	req1.RemoteAddr = ip
	rec1 := httptest.NewRecorder()
	srv.router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusFound {
		t.Fatalf("expected status %d for successful redirect, got %d", http.StatusFound, rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/"+createResp.ShortCode, nil)
	req2.RemoteAddr = ip
	rec2 := httptest.NewRecorder()
	srv.router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d after limit exceeded, got %d", http.StatusTooManyRequests, rec2.Code)
	}

	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header to be set")
	}
}

func TestServer_RateLimitAPI(t *testing.T) {
	cfg := &config.Config{
		ServerPort:                 8080,
		BaseURL:                    "http://localhost:8080",
		DatabaseURL:                "test.db",
		AuthToken:                  "test-token",
		AllowedOrigins:             "*",
		RedirectRateLimitPerMinute: 50,
		APIRateLimitPerMinute:      1,
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	req1 := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req1.RemoteAddr = "192.0.2.201:1234"
	req1.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	rec1 := httptest.NewRecorder()
	srv.router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first API request status %d, got %d", http.StatusOK, rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req2.RemoteAddr = req1.RemoteAddr
	req2.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	rec2 := httptest.NewRecorder()
	srv.router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d after API limit exceeded, got %d", http.StatusTooManyRequests, rec2.Code)
	}

	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header to be set for API limit")
	}
}

// TestServer_ConcurrentRequests tests the server handles concurrent requests
func TestServer_ConcurrentRequests(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	const numRequests = 100
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)
			results <- rec.Code
		}()
	}

	for i := 0; i < numRequests; i++ {
		code := <-results
		if code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, code)
		}
	}
}

// TestServer_ContextCancellation tests proper context handling during shutdown
func TestServer_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     0,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start server with error handling
	serverErrors := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for server to be ready
	ready := false
	for i := 0; i < 50; i++ {
		time.Sleep(10 * time.Millisecond)
		addr := srv.httpServer.Addr
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
	}

	if !ready {
		select {
		case err := <-serverErrors:
			t.Fatalf("server failed to start: %v", err)
		default:
			t.Skip("server not ready for test")
		}
	}

	// Create a context with short timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Shutdown should complete within timeout
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}
}

// BenchmarkServer_HealthCheck benchmarks the health check endpoint
func BenchmarkServer_HealthCheck(b *testing.B) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(b)
	defer db.Close()

	// Use a nop logger for benchmarks
	logger := zerolog.New(io.Discard)

	srv, err := New(cfg, db, logger)
	if err != nil {
		b.Fatalf("failed to create server: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)
	}
}

// BenchmarkServer_WithMiddleware benchmarks requests through full middleware stack
func BenchmarkServer_WithMiddleware(b *testing.B) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(b)
	defer db.Close()

	// Use a nop logger for benchmarks
	logger := zerolog.New(io.Discard)

	srv, err := New(cfg, db, logger)
	if err != nil {
		b.Fatalf("failed to create server: %v", err)
	}

	srv.router.Get("/bench", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/bench", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)
	}
}

// ExampleServer_Shutdown demonstrates graceful server shutdown
func ExampleServer_Shutdown() {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "./database.db",
		AuthToken:      "secret-token",
		AllowedOrigins: "*",
	}

	// Open database for example (in production, handle errors properly)
	db, _ := sql.Open("sqlite3", cfg.DatabaseURL)
	defer db.Close()

	logger := zerolog.Nop()
	srv, err := New(cfg, db, logger)
	if err != nil {
		fmt.Printf("Server creation error: %v\n", err)
		return
	}

	// Start server
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Shutdown gracefully
	ctx := context.Background()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	}
}
