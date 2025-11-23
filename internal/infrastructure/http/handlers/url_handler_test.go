package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// Mock use cases for testing
type mockCreateURLUseCase struct {
	executeFunc func(ctx context.Context, req application.CreateURLRequest) (*application.CreateURLResponse, error)
}

func (m *mockCreateURLUseCase) Execute(ctx context.Context, req application.CreateURLRequest) (*application.CreateURLResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, req)
	}
	return nil, nil
}

type mockListURLsUseCase struct {
	executeFunc func(ctx context.Context, req application.ListURLsRequest) (*application.ListURLsResponse, error)
}

func (m *mockListURLsUseCase) Execute(ctx context.Context, req application.ListURLsRequest) (*application.ListURLsResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, req)
	}
	return nil, nil
}

type mockDeleteURLUseCase struct {
	executeFunc func(ctx context.Context, req application.DeleteURLRequest) (*application.DeleteURLResponse, error)
}

func (m *mockDeleteURLUseCase) Execute(ctx context.Context, req application.DeleteURLRequest) (*application.DeleteURLResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, req)
	}
	return nil, nil
}

// Helper function to add user ID to context
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, userID)
}

// TestURLHandler_Create tests the Create endpoint
func TestURLHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		userID         string
		hasUserID      bool
		mockResponse   *application.CreateURLResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "successful creation",
			requestBody: `{"original_url":"https://example.com"}`,
			userID:      "test-user",
			hasUserID:   true,
			mockResponse: &application.CreateURLResponse{
				ShortCode:   "abc123",
				ShortURL:    "http://localhost:8080/abc123",
				OriginalURL: "https://example.com",
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"short_code":"abc123","short_url":"http://localhost:8080/abc123","original_url":"https://example.com"}`,
		},
		{
			name:           "missing user ID",
			requestBody:    `{"original_url":"https://example.com"}`,
			hasUserID:      false,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid json}`,
			userID:         "test-user",
			hasUserID:      true,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid request body"}`,
		},
		{
			name:           "empty original URL",
			requestBody:    `{"original_url":""}`,
			userID:         "test-user",
			hasUserID:      true,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"original_url is required"}`,
		},
		{
			name:           "missing original URL field",
			requestBody:    `{}`,
			userID:         "test-user",
			hasUserID:      true,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"original_url is required"}`,
		},
		{
			name:           "invalid URL format",
			requestBody:    `{"original_url":"invalid-url"}`,
			userID:         "test-user",
			hasUserID:      true,
			mockError:      url.ErrInvalidOriginalURL,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid original URL format"}`,
		},
		{
			name:           "duplicate short code",
			requestBody:    `{"original_url":"https://example.com"}`,
			userID:         "test-user",
			hasUserID:      true,
			mockError:      url.ErrDuplicateShortCode,
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"short code already exists"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCreate := &mockCreateURLUseCase{
				executeFunc: func(ctx context.Context, req application.CreateURLRequest) (*application.CreateURLResponse, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			handler := NewURLHandler(mockCreate, nil, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.hasUserID {
				req = req.WithContext(withUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			body := rec.Body.String()
			body = body[:len(body)-1] // Remove trailing newline

			if body != tt.expectedBody {
				t.Errorf("expected body %s, got %s", tt.expectedBody, body)
			}
		})
	}
}

// TestURLHandler_List tests the List endpoint
func TestURLHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		hasUserID      bool
		queryParams    string
		mockResponse   *application.ListURLsResponse
		mockError      error
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "successful list",
			userID:      "test-user",
			hasUserID:   true,
			queryParams: "",
			mockResponse: &application.ListURLsResponse{
				URLs: []application.URLResponse{
					{
						ShortCode:   "abc123",
						OriginalURL: "https://example.com",
					},
				},
				Total:  1,
				Limit:  20,
				Offset: 0,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp application.ListURLsResponse
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.URLs) != 1 {
					t.Errorf("expected 1 URL, got %d", len(resp.URLs))
				}
				if resp.URLs[0].ShortCode != "abc123" {
					t.Errorf("expected short code 'abc123', got %s", resp.URLs[0].ShortCode)
				}
			},
		},
		{
			name:           "missing user ID",
			hasUserID:      false,
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string) {
				if body != `{"error":"unauthorized"}`+"\n" {
					t.Errorf("unexpected body: %s", body)
				}
			},
		},
		{
			name:        "with pagination params",
			userID:      "test-user",
			hasUserID:   true,
			queryParams: "?limit=10&offset=5",
			mockResponse: &application.ListURLsResponse{
				URLs:   []application.URLResponse{},
				Total:  0,
				Limit:  10,
				Offset: 5,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp application.ListURLsResponse
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Limit != 10 {
					t.Errorf("expected limit 10, got %d", resp.Limit)
				}
				if resp.Offset != 5 {
					t.Errorf("expected offset 5, got %d", resp.Offset)
				}
			},
		},
		{
			name:        "invalid pagination params use defaults",
			userID:      "test-user",
			hasUserID:   true,
			queryParams: "?limit=invalid&offset=invalid",
			mockResponse: &application.ListURLsResponse{
				URLs:   []application.URLResponse{},
				Total:  0,
				Limit:  20,
				Offset: 0,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp application.ListURLsResponse
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Limit != 20 {
					t.Errorf("expected default limit 20, got %d", resp.Limit)
				}
				if resp.Offset != 0 {
					t.Errorf("expected default offset 0, got %d", resp.Offset)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockList := &mockListURLsUseCase{
				executeFunc: func(ctx context.Context, req application.ListURLsRequest) (*application.ListURLsResponse, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			handler := NewURLHandler(nil, mockList, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/urls"+tt.queryParams, nil)

			if tt.hasUserID {
				req = req.WithContext(withUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()

			handler.List(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestURLHandler_Delete tests the Delete endpoint
func TestURLHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		hasUserID      bool
		shortCode      string
		mockResponse   *application.DeleteURLResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "successful deletion",
			userID:    "test-user",
			hasUserID: true,
			shortCode: "abc123",
			mockResponse: &application.DeleteURLResponse{
				Success: true,
			},
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name:           "missing user ID",
			hasUserID:      false,
			shortCode:      "abc123",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "URL not found",
			userID:         "test-user",
			hasUserID:      true,
			shortCode:      "notfound",
			mockError:      url.ErrURLNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"url not found"}`,
		},
		{
			name:           "unauthorized deletion",
			userID:         "test-user",
			hasUserID:      true,
			shortCode:      "abc123",
			mockError:      url.ErrUnauthorizedDeletion,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"unauthorized: you can only delete URLs you created"}`,
		},
		{
			name:           "invalid short code",
			userID:         "test-user",
			hasUserID:      true,
			shortCode:      "ab",
			mockError:      url.ErrInvalidShortCode,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"short code must be 3-20 characters long and contain only alphanumeric characters, underscores, or hyphens"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDelete := &mockDeleteURLUseCase{
				executeFunc: func(ctx context.Context, req application.DeleteURLRequest) (*application.DeleteURLResponse, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			handler := NewURLHandler(nil, nil, mockDelete)

			req := httptest.NewRequest(http.MethodDelete, "/api/urls/"+tt.shortCode, nil)

			// Set up chi URL params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("shortCode", tt.shortCode)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			if tt.hasUserID {
				req = req.WithContext(withUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()

			handler.Delete(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			body := rec.Body.String()
			if tt.expectedBody != "" {
				body = body[:len(body)-1] // Remove trailing newline
				if body != tt.expectedBody {
					t.Errorf("expected body %s, got %s", tt.expectedBody, body)
				}
			}
		})
	}
}

// TestHandleUseCaseError tests error mapping
func TestHandleUseCaseError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "URL not found",
			err:            url.ErrURLNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"url not found"}`,
		},
		{
			name:           "duplicate short code",
			err:            url.ErrDuplicateShortCode,
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"short code already exists"}`,
		},
		{
			name:           "invalid short code",
			err:            url.ErrInvalidShortCode,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized deletion",
			err:            url.ErrUnauthorizedDeletion,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "unknown error",
			err:            errors.New("unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handleDomainError(rec, tt.err)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedBody != "" {
				body := rec.Body.String()
				body = body[:len(body)-1] // Remove trailing newline
				if body != tt.expectedBody {
					t.Errorf("expected body %s, got %s", tt.expectedBody, body)
				}
			}
		})
	}
}

// TestParseQueryInt tests query parameter parsing
func TestParseQueryInt(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			queryString:  "limit=10",
			key:          "limit",
			defaultValue: 20,
			expected:     10,
		},
		{
			name:         "missing parameter",
			queryString:  "",
			key:          "limit",
			defaultValue: 20,
			expected:     20,
		},
		{
			name:         "invalid integer",
			queryString:  "limit=invalid",
			key:          "limit",
			defaultValue: 20,
			expected:     20,
		},
		{
			name:         "negative integer",
			queryString:  "offset=-5",
			key:          "offset",
			defaultValue: 0,
			expected:     -5, // Parser allows negative, use case should handle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			result := parseQueryInt(req, tt.key, tt.defaultValue)

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
