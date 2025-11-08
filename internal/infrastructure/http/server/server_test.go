package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
)

func TestNew_CreatesServerWithMiddleware(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
	cfg := &config.Config{
		ServerPort:     0, // Use random available port
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

	// Start server in background
	go func() {
		srv.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}
}

func TestServer_Timeouts(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
			cfg := &config.Config{
				ServerPort:     tt.port,
				DatabaseURL:    "test.db",
				AuthToken:      "test-token",
				AllowedOrigins: "*",
			}

			srv := New(cfg)

			if srv.httpServer.Addr != tt.want {
				t.Errorf("expected address %s, got %s", tt.want, srv.httpServer.Addr)
			}
		})
	}
}

func TestServer_Router(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

	router := srv.Router()
	if router == nil {
		t.Error("expected router to be returned")
	}

	if router != srv.router {
		t.Error("expected same router instance to be returned")
	}
}

func ExampleServer_Start() {
	cfg := &config.Config{
		ServerPort:     8080,
		DatabaseURL:    "./database.db",
		AuthToken:      "secret-token",
		AllowedOrigins: "*",
	}

	srv := New(cfg)

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
