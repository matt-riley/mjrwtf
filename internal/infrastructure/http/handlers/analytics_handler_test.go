package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// Mock GetAnalyticsUseCase
type mockGetAnalyticsUseCase struct {
	executeFunc func(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error)
}

func (m *mockGetAnalyticsUseCase) Execute(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

// Helper function to add user ID to context
func withUserIDForAnalytics(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, userID)
}

func TestAnalyticsHandler_GetAnalytics_Success(t *testing.T) {
	mockResp := &application.GetAnalyticsResponse{
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		TotalClicks: 150,
		ByCountry: map[string]int64{
			"US": 100,
			"UK": 50,
		},
		ByReferrer: map[string]int64{
			"https://google.com":  80,
			"https://twitter.com": 70,
		},
		ByDate: map[string]int64{
			"2025-11-20": 50,
			"2025-11-21": 60,
			"2025-11-22": 40,
		},
	}

	useCase := &mockGetAnalyticsUseCase{
		executeFunc: func(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error) {
			if req.ShortCode == "abc123" && req.RequestedBy == "test-user" {
				return mockResp, nil
			}
			return nil, url.ErrURLNotFound
		},
	}

	handler := NewAnalyticsHandler(useCase)

	// Create router with chi
	r := chi.NewRouter()
	r.Get("/api/urls/{shortCode}/analytics", handler.GetAnalytics)

	// Create request with user context
	req := httptest.NewRequest(http.MethodGet, "/api/urls/abc123/analytics", nil)
	req = req.WithContext(withUserIDForAnalytics(req.Context(), "test-user"))
	w := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var resp application.GetAnalyticsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response
	if resp.ShortCode != "abc123" {
		t.Errorf("expected short_code abc123, got %s", resp.ShortCode)
	}

	if resp.TotalClicks != 150 {
		t.Errorf("expected 150 total clicks, got %d", resp.TotalClicks)
	}

	if resp.ByCountry["US"] != 100 {
		t.Errorf("expected 100 US clicks, got %d", resp.ByCountry["US"])
	}
}

func TestAnalyticsHandler_GetAnalytics_WithTimeRange(t *testing.T) {
	startTime := time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 11, 22, 23, 59, 59, 0, time.UTC)

	mockResp := &application.GetAnalyticsResponse{
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		TotalClicks: 100,
		ByCountry: map[string]int64{
			"US": 70,
			"UK": 30,
		},
		ByReferrer: map[string]int64{
			"https://google.com": 60,
		},
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	useCase := &mockGetAnalyticsUseCase{
		executeFunc: func(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error) {
			if req.ShortCode == "abc123" && req.RequestedBy == "test-user" && req.StartTime != nil && req.EndTime != nil {
				return mockResp, nil
			}
			return nil, url.ErrURLNotFound
		},
	}

	handler := NewAnalyticsHandler(useCase)

	// Create router with chi
	r := chi.NewRouter()
	r.Get("/api/urls/{shortCode}/analytics", handler.GetAnalytics)

	// Create request with time range and user context
	req := httptest.NewRequest(http.MethodGet, "/api/urls/abc123/analytics?start_time=2025-11-20T00:00:00Z&end_time=2025-11-22T23:59:59Z", nil)
	req = req.WithContext(withUserIDForAnalytics(req.Context(), "test-user"))
	w := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var resp application.GetAnalyticsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response
	if resp.TotalClicks != 100 {
		t.Errorf("expected 100 total clicks, got %d", resp.TotalClicks)
	}

	if resp.StartTime == nil || resp.EndTime == nil {
		t.Error("expected start_time and end_time to be present")
	}
}

func TestAnalyticsHandler_GetAnalytics_NotFound(t *testing.T) {
	useCase := &mockGetAnalyticsUseCase{
		executeFunc: func(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error) {
			return nil, url.ErrURLNotFound
		},
	}

	handler := NewAnalyticsHandler(useCase)

	// Create router with chi
	r := chi.NewRouter()
	r.Get("/api/urls/{shortCode}/analytics", handler.GetAnalytics)

	// Create request with user context
	req := httptest.NewRequest(http.MethodGet, "/api/urls/notfound/analytics", nil)
	req = req.WithContext(withUserIDForAnalytics(req.Context(), "test-user"))
	w := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetAnalytics_Unauthorized(t *testing.T) {
	useCase := &mockGetAnalyticsUseCase{
		executeFunc: func(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error) {
			return nil, url.ErrUnauthorizedDeletion
		},
	}

	handler := NewAnalyticsHandler(useCase)

	// Create router with chi
	r := chi.NewRouter()
	r.Get("/api/urls/{shortCode}/analytics", handler.GetAnalytics)

	// Create request with user context
	req := httptest.NewRequest(http.MethodGet, "/api/urls/abc123/analytics", nil)
	req = req.WithContext(withUserIDForAnalytics(req.Context(), "test-user"))
	w := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(w, req)

	// Check status code - should be 403 Forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetAnalytics_NoUserInContext(t *testing.T) {
	useCase := &mockGetAnalyticsUseCase{
		executeFunc: func(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error) {
			return nil, errors.New("should not be called")
		},
	}

	handler := NewAnalyticsHandler(useCase)

	// Create router with chi
	r := chi.NewRouter()
	r.Get("/api/urls/{shortCode}/analytics", handler.GetAnalytics)

	// Create request WITHOUT user context
	req := httptest.NewRequest(http.MethodGet, "/api/urls/abc123/analytics", nil)
	w := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(w, req)

	// Check status code - should be 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetAnalytics_InvalidTimeFormat(t *testing.T) {
	useCase := &mockGetAnalyticsUseCase{
		executeFunc: func(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error) {
			return nil, errors.New("should not be called")
		},
	}

	handler := NewAnalyticsHandler(useCase)

	// Create router with chi
	r := chi.NewRouter()
	r.Get("/api/urls/{shortCode}/analytics", handler.GetAnalytics)

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "invalid start_time format",
			url:  "/api/urls/abc123/analytics?start_time=2025-11-20&end_time=2025-11-22T23:59:59Z",
		},
		{
			name: "invalid end_time format",
			url:  "/api/urls/abc123/analytics?start_time=2025-11-20T00:00:00Z&end_time=2025-11-22",
		},
		{
			name: "missing end_time",
			url:  "/api/urls/abc123/analytics?start_time=2025-11-20T00:00:00Z",
		},
		{
			name: "missing start_time",
			url:  "/api/urls/abc123/analytics?end_time=2025-11-22T23:59:59Z",
		},
		{
			name: "start_time after end_time",
			url:  "/api/urls/abc123/analytics?start_time=2025-11-22T23:59:59Z&end_time=2025-11-20T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = req.WithContext(withUserIDForAnalytics(req.Context(), "test-user"))
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}
