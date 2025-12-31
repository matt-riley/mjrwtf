package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

func ptrString(s string) *string { return &s }

// Mock redirect use case for testing
type mockRedirectUseCase struct {
	executeFunc func(ctx context.Context, req application.RedirectRequest) (*application.RedirectResponse, error)
}

func (m *mockRedirectUseCase) Execute(ctx context.Context, req application.RedirectRequest) (*application.RedirectResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, req)
	}
	return nil, nil
}

// TestRedirectHandler_Redirect tests the Redirect endpoint
func TestRedirectHandler_Redirect(t *testing.T) {
	tests := []struct {
		name             string
		shortCode        string
		referrer         string
		userAgent        string
		mockResponse     *application.RedirectResponse
		mockError        error
		expectedStatus   int
		expectedLocation string
		checkRequestData func(t *testing.T, req application.RedirectRequest)
	}{
		{
			name:      "successful redirect",
			shortCode: "abc123",
			referrer:  "https://google.com",
			userAgent: "Mozilla/5.0",
			mockResponse: &application.RedirectResponse{
				OriginalURL: "https://example.com",
			},
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com",
			checkRequestData: func(t *testing.T, req application.RedirectRequest) {
				if req.ShortCode != "abc123" {
					t.Errorf("expected shortCode 'abc123', got '%s'", req.ShortCode)
				}
				if req.Referrer != "https://google.com" {
					t.Errorf("expected referrer 'https://google.com', got '%s'", req.Referrer)
				}
				if req.UserAgent != "Mozilla/5.0" {
					t.Errorf("expected userAgent 'Mozilla/5.0', got '%s'", req.UserAgent)
				}
			},
		},
		{
			name:      "redirect without referrer",
			shortCode: "abc123",
			referrer:  "",
			userAgent: "Mozilla/5.0",
			mockResponse: &application.RedirectResponse{
				OriginalURL: "https://example.com",
			},
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com",
			checkRequestData: func(t *testing.T, req application.RedirectRequest) {
				if req.Referrer != "" {
					t.Errorf("expected empty referrer, got '%s'", req.Referrer)
				}
			},
		},
		{
			name:      "redirect without user agent",
			shortCode: "abc123",
			referrer:  "https://google.com",
			userAgent: "",
			mockResponse: &application.RedirectResponse{
				OriginalURL: "https://example.com",
			},
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com",
			checkRequestData: func(t *testing.T, req application.RedirectRequest) {
				if req.UserAgent != "" {
					t.Errorf("expected empty userAgent, got '%s'", req.UserAgent)
				}
			},
		},
		{
			name:           "short code not found",
			shortCode:      "notfound",
			mockError:      url.ErrURLNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "internal server error",
			shortCode:      "abc123",
			mockError:      errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "gone url renders interstitial",
			shortCode: "gone123",
			mockResponse: &application.RedirectResponse{
				OriginalURL:      "https://example.com/gone",
				IsGone:          true,
				GoneStatusCode:  http.StatusGone,
				ArchiveURL:      ptrString("https://web.archive.org/web/20200101000000/https://example.com/gone"),
			},
			expectedStatus: http.StatusGone,
		},
		{
			name:      "redirect preserves original URL intact",
			shortCode: "abc123",
			mockResponse: &application.RedirectResponse{
				OriginalURL: "https://example.com/path?query=value&foo=bar",
			},
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com/path?query=value&foo=bar",
		},
		{
			name:      "redirect with special characters in URL",
			shortCode: "abc123",
			mockResponse: &application.RedirectResponse{
				OriginalURL: "https://example.com/path?q=hello%20world&special=%26%3D%3F",
			},
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com/path?q=hello%20world&special=%26%3D%3F",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest application.RedirectRequest
			mockRedirect := &mockRedirectUseCase{
				executeFunc: func(ctx context.Context, req application.RedirectRequest) (*application.RedirectResponse, error) {
					capturedRequest = req
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			handler := NewRedirectHandler(mockRedirect)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortCode, nil)
			if tt.referrer != "" {
				req.Header.Set("Referer", tt.referrer)
			}
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			// Set up chi URL params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("shortCode", tt.shortCode)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()

			handler.Redirect(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedLocation != "" {
				location := rec.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected location '%s', got '%s'", tt.expectedLocation, location)
				}
			}

			if tt.mockResponse != nil && tt.mockResponse.IsGone {
				if ct := rec.Header().Get("Content-Type"); ct == "" {
					t.Error("expected Content-Type to be set")
				}
				if body := rec.Body.String(); body == "" {
					t.Error("expected interstitial HTML body")
				}
			}

			if tt.checkRequestData != nil {
				tt.checkRequestData(t, capturedRequest)
			}
		})
	}
}

// TestRedirectHandler_EmptyShortCode tests handling of empty short codes
func TestRedirectHandler_EmptyShortCode(t *testing.T) {
	mockRedirect := &mockRedirectUseCase{
		executeFunc: func(ctx context.Context, req application.RedirectRequest) (*application.RedirectResponse, error) {
			t.Error("use case should not be called for empty short code")
			return nil, nil
		},
	}

	handler := NewRedirectHandler(mockRedirect)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Set up chi URL params with empty short code
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortCode", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()

	handler.Redirect(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d for empty short code, got %d", http.StatusNotFound, rec.Code)
	}
}
