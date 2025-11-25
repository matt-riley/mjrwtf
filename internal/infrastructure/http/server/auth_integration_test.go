package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

func TestServer_AuthMiddlewareIntegration(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-secret-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Add a test protected route to demonstrate auth middleware usage
	srv.router.Route("/api/protected", func(r chi.Router) {
		r.Use(middleware.Auth(cfg.AuthToken))
		r.Get("/resource", func(w http.ResponseWriter, r *http.Request) {
			// Demonstrate extracting user ID from context
			userID, ok := middleware.GetUserID(r.Context())
			if !ok {
				t.Error("expected user ID to be in context")
			}
			if userID != "authenticated-user" {
				t.Errorf("expected user ID %q, got %q", "authenticated-user", userID)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("access granted"))
		})
	})

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "{\"error\":\"Unauthorized: missing authorization header\"}\n",
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "{\"error\":\"Unauthorized: invalid token\"}\n",
		},
		{
			name:           "valid token",
			authHeader:     "Bearer test-secret-token",
			expectedStatus: http.StatusOK,
			expectedBody:   "access granted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/protected/resource", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestServer_UnprotectedRouteWithoutAuth(t *testing.T) {
	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-secret-token",
		AllowedOrigins: "*",
	}

	db := setupTestDB(t)
	defer db.Close()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Health check should work without authentication
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for unprotected health check, got %d", http.StatusOK, rec.Code)
	}

	expected := `{"status":"ok"}`
	if rec.Body.String() != expected {
		t.Errorf("expected response %s, got %s", expected, rec.Body.String())
	}
}
