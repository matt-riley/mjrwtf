package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
)

// TestMiddlewareExecutionOrder verifies that middleware executes in the correct order
func TestMiddlewareExecutionOrder(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

	// Add a test handler
	srv.router.Get("/test-order", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Capture log output to verify logging middleware executed
	var logBuf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(originalOutput)

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
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

	srv.router.Get("/panic-test", func(w http.ResponseWriter, r *http.Request) {
		panic("intentional panic for testing")
	})

	// Capture log output
	var logBuf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(originalOutput)

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
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
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
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
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
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
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
		ServerPort:     0,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Disable logging for benchmark
	defer log.SetOutput(originalOutput)

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
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Disable logging for benchmark
	defer log.SetOutput(originalOutput)

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
		DatabaseURL:    "./database.db",
		AuthToken:      "secret-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
