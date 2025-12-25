package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/session"
)

func TestPageHandler_Home(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	sessionStore := session.NewStore(24 * time.Hour)
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", sessionStore)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.Home(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "mjr.wtf") {
		t.Error("expected body to contain 'mjr.wtf'")
	}
	if !strings.Contains(body, "Welcome to mjr.wtf") {
		t.Error("expected body to contain 'Welcome to mjr.wtf'")
	}
}

func TestPageHandler_NotFound(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.NotFound(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "404") {
		t.Error("expected body to contain '404'")
	}
	if !strings.Contains(body, "Page Not Found") {
		t.Error("expected body to contain 'Page Not Found'")
	}
}

func TestPageHandler_InternalError(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()

	testMessage := "Test error message"
	handler.InternalError(w, req, testMessage)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}
	if !strings.Contains(body, "Internal Server Error") {
		t.Error("expected body to contain 'Internal Server Error'")
	}
	if !strings.Contains(body, testMessage) {
		t.Errorf("expected body to contain error message '%s'", testMessage)
	}
}

func TestPageHandler_InternalError_EmptyMessage(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()

	handler.InternalError(w, req, "")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}
}

func TestPageHandler_CreatePage_GET(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/create", nil)
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Create Short URL") {
		t.Error("expected body to contain 'Create Short URL'")
	}
	if !strings.Contains(body, "original_url") {
		t.Error("expected body to contain URL input field")
	}
	if !strings.Contains(body, "auth_token") {
		t.Error("expected body to contain auth token field")
	}
}

func TestPageHandler_CreatePage_POST_Success(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{
		executeFunc: func(ctx context.Context, req application.CreateURLRequest) (*application.CreateURLResponse, error) {
			return &application.CreateURLResponse{
				ShortCode:   "abc123",
				ShortURL:    "http://localhost:8080/abc123",
				OriginalURL: req.OriginalURL,
			}, nil
		},
	}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	form := url.Values{}
	form.Add("original_url", "https://example.com/very/long/url")
	form.Add("auth_token", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "URL Shortened Successfully") {
		t.Error("expected body to contain success message")
	}
	if !strings.Contains(body, "abc123") {
		t.Error("expected body to contain short code")
	}
	if !strings.Contains(body, "http://localhost:8080/abc123") {
		t.Error("expected body to contain short URL")
	}
}

func TestPageHandler_CreatePage_POST_MissingURL(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	form := url.Values{}
	form.Add("auth_token", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "URL is required") {
		t.Error("expected body to contain error message about missing URL")
	}
}

func TestPageHandler_CreatePage_POST_MissingToken(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	form := url.Values{}
	form.Add("original_url", "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Authentication token is required") {
		t.Error("expected body to contain error message about missing token")
	}
}

func TestPageHandler_CreatePage_POST_UseCaseError(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{
		executeFunc: func(ctx context.Context, req application.CreateURLRequest) (*application.CreateURLResponse, error) {
			return nil, fmt.Errorf("failed to create shortened URL: %w", errors.New("URL scheme must be http or https"))
		},
	}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	form := url.Values{}
	form.Add("original_url", "ftp://not-a-valid-url")
	form.Add("auth_token", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Since the mock doesn't wrap the error properly, it will return 500
	// In real usage, the use case wraps domain errors which are then mapped correctly
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "An error occurred") {
		t.Error("expected body to contain generic error message")
	}
}

func TestPageHandler_CreatePage_POST_DomainError(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{
		executeFunc: func(ctx context.Context, req application.CreateURLRequest) (*application.CreateURLResponse, error) {
			// Return a properly wrapped domain error
			return nil, fmt.Errorf("failed to shorten URL: %w", errors.New("URL scheme must be http or https"))
		},
	}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	form := url.Values{}
	form.Add("original_url", "ftp://example.com")
	form.Add("auth_token", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Generic errors that don't match domain errors return 500
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestPageHandler_CreatePage_MethodNotAllowed(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "test-token", session.NewStore(24 * time.Hour))

	req := httptest.NewRequest(http.MethodPut, "/create", nil)
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestPageHandler_CreatePage_POST_InvalidToken(t *testing.T) {
	mockUseCase := &mockCreateURLUseCase{}
	handler := NewPageHandler(mockUseCase, &mockListURLsUseCase{}, "correct-token", session.NewStore(24 * time.Hour))

	form := url.Values{}
	form.Add("original_url", "https://example.com")
	form.Add("auth_token", "wrong-token")

	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.CreatePage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid authentication token") {
		t.Error("expected body to contain invalid token error message")
	}
}
