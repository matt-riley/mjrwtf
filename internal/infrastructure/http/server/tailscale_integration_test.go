package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// Integration tests for Tailscale authentication mode.
// These tests verify the end-to-end behavior of both Tailscale and standard modes.

// mockWhoIsClientForIntegration is a configurable mock for integration testing.
type mockWhoIsClientForIntegration struct {
	Profile  *middleware.TailscaleUserProfile
	Err      error
	CallLog  []string // Records remote addresses passed to WhoIs
}

func (m *mockWhoIsClientForIntegration) WhoIs(ctx context.Context, remoteAddr string) (*middleware.TailscaleUserProfile, error) {
	m.CallLog = append(m.CallLog, remoteAddr)
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Profile, nil
}

// TestTailscaleMode_PublicRoutesAccessible verifies that public routes are accessible
// without authentication in Tailscale mode.
func TestTailscaleMode_PublicRoutesAccessible(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = true

	mockClient := &mockWhoIsClientForIntegration{
		Profile: &middleware.TailscaleUserProfile{
			LoginName:   "alice@example.com",
			DisplayName: "Alice",
			NodeName:    "alice-laptop",
		},
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	publicRoutes := []struct {
		name   string
		path   string
		method string
	}{
		{"home page", "/", http.MethodGet},
		{"health check", "/health", http.MethodGet},
		{"ready check", "/ready", http.MethodGet},
		{"create page", "/create", http.MethodGet},
	}

	for _, route := range publicRoutes {
		t.Run(route.name, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}

			// Verify WhoIs was NOT called for public routes
			if len(mockClient.CallLog) > 0 {
				t.Errorf("WhoIs should not be called for public route %s", route.path)
			}
		})
		// Reset call log between tests
		mockClient.CallLog = nil
	}
}

// TestTailscaleMode_AdminRoutesProtected verifies that admin routes require valid
// Tailscale authentication.
func TestTailscaleMode_AdminRoutesProtected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = true

	mockClient := &mockWhoIsClientForIntegration{
		Profile: &middleware.TailscaleUserProfile{
			LoginName:   "alice@example.com",
			DisplayName: "Alice",
			NodeName:    "alice-laptop",
		},
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	adminRoutes := []struct {
		name   string
		path   string
		method string
	}{
		{"dashboard", "/dashboard", http.MethodGet},
		{"API list URLs", "/api/urls", http.MethodGet},
	}

	for _, route := range adminRoutes {
		t.Run(route.name, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200 for authenticated request, got %d", rec.Code)
			}

			// Verify WhoIs WAS called for admin routes
			if len(mockClient.CallLog) == 0 {
				t.Errorf("WhoIs should be called for admin route %s", route.path)
			}
		})
		// Reset call log between tests
		mockClient.CallLog = nil
	}
}

// TestStandardMode_FallbackAuth verifies that standard Bearer/Session auth works
// when Tailscale is disabled.
func TestStandardMode_FallbackAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = false

	// No Tailscale client
	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("API requires auth without Bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("API accessible with valid Bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("dashboard redirects to login without session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("expected redirect (303), got %d", rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "/login" {
			t.Errorf("expected redirect to /login, got %s", location)
		}
	})

	t.Run("login page accessible", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

// TestTailscaleMode_WhoIsFailure verifies that admin routes return 401 when
// WhoIs lookup fails.
func TestTailscaleMode_WhoIsFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = true

	// Mock WhoIs that always fails
	mockClient := &mockWhoIsClientForIntegration{
		Err: errors.New("connection refused"),
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	adminRoutes := []struct {
		path   string
		method string
	}{
		{"/dashboard", http.MethodGet},
		{"/api/urls", http.MethodGet},
	}

	for _, route := range adminRoutes {
		t.Run(route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 when WhoIs fails, got %d", rec.Code)
			}
		})
	}
}

// TestBothModesCanCoexistInCodebase verifies that switching between modes
// works correctly without code changes.
func TestBothModesCanCoexistInCodebase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	// Test 1: Standard mode
	t.Run("standard mode", func(t *testing.T) {
		cfg.TailscaleEnabled = false
		srv, err := New(cfg, db, testLogger())
		if err != nil {
			t.Fatalf("failed to create server in standard mode: %v", err)
		}

		// Verify standard auth is active
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("standard mode: expected 401 without auth, got %d", rec.Code)
		}

		// With auth
		req = httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
		rec = httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("standard mode: expected 200 with auth, got %d", rec.Code)
		}
	})

	// Test 2: Tailscale mode
	t.Run("tailscale mode", func(t *testing.T) {
		cfg.TailscaleEnabled = true
		mockClient := &mockWhoIsClientForIntegration{
			Profile: &middleware.TailscaleUserProfile{
				LoginName: "alice@example.com",
			},
		}

		srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
		if err != nil {
			t.Fatalf("failed to create server in Tailscale mode: %v", err)
		}

		// Verify Tailscale auth is active (no Bearer required)
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("tailscale mode: expected 200 (WhoIs provides auth), got %d", rec.Code)
		}
	})
}

// TestTailscaleMode_LoginRouteHidden verifies that login/logout routes are
// not available in Tailscale mode.
func TestTailscaleMode_LoginRouteHidden(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = true

	mockClient := &mockWhoIsClientForIntegration{
		Profile: &middleware.TailscaleUserProfile{
			LoginName: "alice@example.com",
		},
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	hiddenRoutes := []string{"/login", "/logout"}

	for _, path := range hiddenRoutes {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			// Login/logout routes should return 404 in Tailscale mode
			if rec.Code != http.StatusNotFound {
				t.Errorf("expected 404 for %s in Tailscale mode, got %d", path, rec.Code)
			}
		})
	}
}

// TestTailscaleMode_UserIdentityInContext verifies that the Tailscale user
// identity is properly propagated to handlers.
func TestTailscaleMode_UserIdentityInContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()
	cfg.TailscaleEnabled = true

	mockClient := &mockWhoIsClientForIntegration{
		Profile: &middleware.TailscaleUserProfile{
			LoginName:   "alice@example.com",
			DisplayName: "Alice Smith",
			NodeName:    "alice-macbook",
		},
	}

	srv, err := New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Make request to dashboard (which uses user identity)
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// The dashboard should render successfully (user identity was available)
	body := rec.Body.String()
	if !strings.Contains(body, "Dashboard") && !strings.Contains(body, "dashboard") {
		t.Error("expected dashboard content to be rendered")
	}
}

// TestTailscaleMode_MetricsEndpointUnaffected verifies that the /metrics endpoint
// behavior is not affected by Tailscale mode.
func TestTailscaleMode_MetricsEndpointUnaffected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCases := []struct {
		name               string
		tailscaleEnabled   bool
		metricsAuthEnabled bool
		authHeader         string
		expectedStatus     int
	}{
		{
			name:               "standard mode - metrics without auth",
			tailscaleEnabled:   false,
			metricsAuthEnabled: false,
			authHeader:         "",
			expectedStatus:     http.StatusOK,
		},
		{
			name:               "standard mode - metrics with auth required",
			tailscaleEnabled:   false,
			metricsAuthEnabled: true,
			authHeader:         "",
			expectedStatus:     http.StatusUnauthorized,
		},
		{
			name:               "tailscale mode - metrics without auth",
			tailscaleEnabled:   true,
			metricsAuthEnabled: false,
			authHeader:         "",
			expectedStatus:     http.StatusOK,
		},
		{
			name:               "tailscale mode - metrics with auth required",
			tailscaleEnabled:   true,
			metricsAuthEnabled: true,
			authHeader:         "",
			expectedStatus:     http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.TailscaleEnabled = tc.tailscaleEnabled
			cfg.MetricsAuthEnabled = tc.metricsAuthEnabled

			var srv *Server
			var err error

			if tc.tailscaleEnabled {
				mockClient := &mockWhoIsClientForIntegration{
					Profile: &middleware.TailscaleUserProfile{LoginName: "test@test.com"},
				}
				srv, err = New(cfg, db, testLogger(), WithTailscaleClient(mockClient))
			} else {
				srv, err = New(cfg, db, testLogger())
			}

			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			srv.router.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}
