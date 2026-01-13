package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// mockWhoIsClient is a mock implementation of middleware.WhoIsClient for testing.
type mockWhoIsClient struct {
	profile *middleware.TailscaleUserProfile
	err     error
}

func (m *mockWhoIsClient) WhoIs(ctx context.Context, remoteAddr string) (*middleware.TailscaleUserProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.profile, nil
}

func TestNew_WithTailscaleClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	mockClient := &mockWhoIsClient{
		profile: &middleware.TailscaleUserProfile{
			LoginName:   "alice@example.com",
			DisplayName: "Alice",
			NodeName:    "alice-laptop",
		},
	}

	// Create server with Tailscale client using functional option
	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if srv == nil {
		t.Fatal("expected server to be created")
	}

	if srv.tailscaleClient != mockClient {
		t.Error("expected tailscale client to be set")
	}
}

func TestNew_WithoutTailscaleClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	// Create server without Tailscale client
	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if srv == nil {
		t.Fatal("expected server to be created")
	}

	if srv.tailscaleClient != nil {
		t.Error("expected tailscale client to be nil")
	}
}

func TestServer_AdminRoutes_TailscaleMode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = true

	mockClient := &mockWhoIsClient{
		profile: &middleware.TailscaleUserProfile{
			LoginName:   "alice@example.com",
			DisplayName: "Alice",
			NodeName:    "alice-laptop",
		},
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
	}{
		{
			name:           "dashboard accessible with Tailscale auth",
			path:           "/dashboard",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API urls accessible with Tailscale auth",
			path:           "/api/urls",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestServer_AdminRoutes_StandardMode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = false

	// No Tailscale client - should use standard Bearer/Session auth
	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		method         string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "API urls requires Bearer auth",
			path:           "/api/urls",
			method:         http.MethodGet,
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "API urls accessible with Bearer token",
			path:           "/api/urls",
			method:         http.MethodGet,
			authHeader:     "Bearer test-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "dashboard redirects to login without session",
			path:           "/dashboard",
			method:         http.MethodGet,
			authHeader:     "",
			expectedStatus: http.StatusSeeOther, // redirect to login
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestServer_PublicRoutes_AccessibleInBothModes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	mockClient := &mockWhoIsClient{
		profile: &middleware.TailscaleUserProfile{
			LoginName:   "alice@example.com",
			DisplayName: "Alice",
			NodeName:    "alice-laptop",
		},
	}

	testCases := []struct {
		name            string
		tailscaleClient middleware.WhoIsClient
	}{
		{
			name:            "standard mode",
			tailscaleClient: nil,
		},
		{
			name:            "tailscale mode",
			tailscaleClient: mockClient,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var srv *Server
			var err error

			if tc.tailscaleClient != nil {
				cfg.TailscaleEnabled = true
				srv, err = New(cfg, db, testLogger(), WithTailscaleClient(tc.tailscaleClient))
			} else {
				cfg.TailscaleEnabled = false
				srv, err = New(cfg, db, testLogger())
			}

			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}

			publicRoutes := []struct {
				path           string
				expectedStatus int
			}{
				{"/", http.StatusOK},
				{"/health", http.StatusOK},
				{"/ready", http.StatusOK},
			}

			for _, route := range publicRoutes {
				req := httptest.NewRequest(http.MethodGet, route.path, nil)
				rec := httptest.NewRecorder()

				srv.router.ServeHTTP(rec, req)

				if rec.Code != route.expectedStatus {
					t.Errorf("route %s: expected status %d, got %d", route.path, route.expectedStatus, rec.Code)
				}
			}
		})
	}
}

func TestServer_TailscaleMode_WhoIsFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = true

	mockClient := &mockWhoIsClient{
		err: context.DeadlineExceeded, // Simulate WhoIs failure
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Admin routes should return 401 when WhoIs fails
	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestServer_TailscaleClient_Getter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	mockClient := &mockWhoIsClient{
		profile: &middleware.TailscaleUserProfile{
			LoginName: "alice@example.com",
		},
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if srv.TailscaleClient() != mockClient {
		t.Error("expected TailscaleClient() to return the mock client")
	}
}

func TestServerOption_WithTailscaleClient(t *testing.T) {
	mockClient := &mockWhoIsClient{}

	opt := WithTailscaleClient(mockClient)
	if opt == nil {
		t.Fatal("expected option to be created")
	}
}

func TestServerMode_Logging(t *testing.T) {
	// This test verifies the server logs which mode it's running in.
	// Since we use zerolog.Nop() in tests, we just verify the server starts correctly.

	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name              string
		tailscaleEnabled  bool
		tailscaleClient   middleware.WhoIsClient
		expectTailscale   bool
	}{
		{
			name:             "standard mode when disabled",
			tailscaleEnabled: false,
			tailscaleClient:  nil,
			expectTailscale:  false,
		},
		{
			name:             "tailscale mode when enabled with client",
			tailscaleEnabled: true,
			tailscaleClient: &mockWhoIsClient{
				profile: &middleware.TailscaleUserProfile{LoginName: "test@test.com"},
			},
			expectTailscale: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.TailscaleEnabled = tt.tailscaleEnabled

			var srv *Server
			var err error

			if tt.tailscaleClient != nil {
				srv, err = New(cfg, db, testLogger(), WithTailscaleClient(tt.tailscaleClient))
			} else {
				srv, err = New(cfg, db, testLogger())
			}

			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}

			hasTailscale := srv.TailscaleClient() != nil
			if hasTailscale != tt.expectTailscale {
				t.Errorf("expected tailscale=%v, got %v", tt.expectTailscale, hasTailscale)
			}
		})
	}
}
