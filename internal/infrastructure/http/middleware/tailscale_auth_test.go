package middleware_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
	"github.com/rs/zerolog"
)

// mockWhoIsClient implements the WhoIsClient interface for testing
type mockWhoIsClient struct {
	userProfile *middleware.TailscaleUserProfile
	err         error
}

func (m *mockWhoIsClient) WhoIs(ctx context.Context, remoteAddr string) (*middleware.TailscaleUserProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.userProfile, nil
}

func TestTailscaleAuth_Success(t *testing.T) {
	client := &mockWhoIsClient{
		userProfile: &middleware.TailscaleUserProfile{
			LoginName:   "alice@example.com",
			DisplayName: "Alice Smith",
			NodeName:    "alice-laptop",
		},
	}
	logger := zerolog.Nop()

	handler := middleware.TailscaleAuth(client, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify TailscaleUser context is set
		profile, ok := middleware.GetTailscaleUser(r.Context())
		if !ok {
			t.Error("Expected Tailscale user in context")
			return
		}
		if profile.LoginName != "alice@example.com" {
			t.Errorf("Expected login 'alice@example.com', got: %s", profile.LoginName)
		}
		if profile.DisplayName != "Alice Smith" {
			t.Errorf("Expected name 'Alice Smith', got: %s", profile.DisplayName)
		}

		// Verify UserID context is also set (for compatibility with Bearer auth handlers)
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			t.Error("Expected UserID to be set in context")
		}
		if userID != "alice@example.com" {
			t.Errorf("Expected UserID 'alice@example.com', got: %s", userID)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req.RemoteAddr = "100.64.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rec.Code)
	}
}

func TestTailscaleAuth_WhoIsFails(t *testing.T) {
	client := &mockWhoIsClient{
		err: net.UnknownNetworkError("connection failed"),
	}
	logger := zerolog.Nop()

	handler := middleware.TailscaleAuth(client, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when WhoIs fails")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req.RemoteAddr = "100.64.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got: %d", rec.Code)
	}
}

func TestTailscaleAuth_NilClient(t *testing.T) {
	logger := zerolog.Nop()

	handler := middleware.TailscaleAuth(nil, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with nil client")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req.RemoteAddr = "100.64.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got: %d", rec.Code)
	}
}

func TestTailscaleAuth_EmptyLoginName(t *testing.T) {
	client := &mockWhoIsClient{
		userProfile: &middleware.TailscaleUserProfile{
			LoginName:   "", // Empty login name should fail validation
			DisplayName: "Unknown User",
			NodeName:    "unknown-node",
		},
	}
	logger := zerolog.Nop()

	handler := middleware.TailscaleAuth(client, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when LoginName is empty")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req.RemoteAddr = "100.64.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got: %d", rec.Code)
	}
}

func TestGetTailscaleUser_NotSet(t *testing.T) {
	ctx := context.Background()
	_, ok := middleware.GetTailscaleUser(ctx)
	if ok {
		t.Error("Expected GetTailscaleUser to return false for empty context")
	}
}

func TestTailscaleUserProfile_UserID(t *testing.T) {
	profile := &middleware.TailscaleUserProfile{
		LoginName:   "alice@example.com",
		DisplayName: "Alice Smith",
	}

	userID := profile.UserID()
	if userID != "alice@example.com" {
		t.Errorf("Expected UserID 'alice@example.com', got: %s", userID)
	}
}
