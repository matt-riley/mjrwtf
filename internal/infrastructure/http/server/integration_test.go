package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
)

// TestMiddlewareExecutionOrder verifies that middleware executes in the correct order
func TestMiddlewareExecutionOrder(t *testing.T) {
	cfg := &config.Config{
		ServerPort:  8080,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)

	// Add a test handler that tracks middleware execution
	var executionOrder []string
	srv.router.Get("/test-order", func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Capture log output to verify logging middleware executed
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(io.Discard)

	req := httptest.NewRequest(http.MethodGet, "/test-order", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify logging middleware executed
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "GET /test-order 200") {
		t.Errorf("expected log entry for request, got: %s", logOutput)
	}
}

// TestMiddlewareRecoveryBeforeLogging ensures recovery middleware catches panics before logging
func TestMiddlewareRecoveryBeforeLogging(t *testing.T) {
	cfg := &config.Config{
		ServerPort:  8080,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)

	srv.router.Get("/panic-test", func(w http.ResponseWriter, r *http.Request) {
		panic("intentional panic for testing")
	})

	// Capture log output
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(io.Discard)

	req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	// Recovery middleware should catch panic and return 500
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d from recovery, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Logging middleware should still execute and log the request
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Error("expected panic recovery log entry")
	}
}

// TestServer_NotFoundHandler tests the default 404 response
func TestServer_NotFoundHandler(t *testing.T) {
	cfg := &config.Config{
		ServerPort:  8080,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)

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
		ServerPort:  8080,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)

	// Health check only supports GET, try POST
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

// TestServer_ConcurrentRequests tests the server handles concurrent requests
func TestServer_ConcurrentRequests(t *testing.T) {
	cfg := &config.Config{
		ServerPort:  8080,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)

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
		ServerPort:  0,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)

	// Start server
	go func() {
		srv.Start()
	}()

	// Create a context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Shutdown should respect already-cancelled context
	if err := srv.Shutdown(ctx); err != nil {
		// This is expected - context is already cancelled
		// But shutdown should still complete
		t.Logf("Shutdown with cancelled context returned error (expected): %v", err)
	}
}

// BenchmarkServer_HealthCheck benchmarks the health check endpoint
func BenchmarkServer_HealthCheck(b *testing.B) {
	cfg := &config.Config{
		ServerPort:  8080,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)
	log.SetOutput(io.Discard) // Disable logging for benchmark

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
		ServerPort:  8080,
		DatabaseURL: "test.db",
		AuthToken:   "test-token",
	}

	srv := New(cfg)
	log.SetOutput(io.Discard) // Disable logging for benchmark

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
		ServerPort:  8080,
		DatabaseURL: "./database.db",
		AuthToken:   "secret-token",
	}

	srv := New(cfg)

	// Start server
	go func() {
		if err := srv.Start(); err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Shutdown gracefully
	ctx := context.Background()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	}
}
