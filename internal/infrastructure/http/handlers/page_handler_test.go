package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageHandler_Home(t *testing.T) {
	handler := NewPageHandler()

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
	handler := NewPageHandler()

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
	handler := NewPageHandler()

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
	handler := NewPageHandler()

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
