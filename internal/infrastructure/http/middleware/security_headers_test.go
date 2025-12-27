package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders_AllHeadersSet(t *testing.T) {
	handler := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check that all required headers are set
	headers := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Content-Security-Policy",
	}

	for _, headerName := range headers {
		if rec.Header().Get(headerName) == "" {
			t.Errorf("expected %s header to be set", headerName)
		}
	}
}

func TestSecurityHeaders_HSTS_Enabled(t *testing.T) {
	handler := SecurityHeaders(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("expected Strict-Transport-Security header when HSTS is enabled")
	}

	expectedHSTS := "max-age=31536000; includeSubDomains"
	if hsts != expectedHSTS {
		t.Errorf("expected HSTS header %q, got %q", expectedHSTS, hsts)
	}
}

func TestSecurityHeaders_HSTS_Disabled(t *testing.T) {
	handler := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("expected no Strict-Transport-Security header when HSTS is disabled, got %q", hsts)
	}
}

func TestSecurityHeaders_ValuesCorrect(t *testing.T) {
	handler := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "X-Content-Type-Options",
			header:   "X-Content-Type-Options",
			expected: "nosniff",
		},
		{
			name:     "X-Frame-Options",
			header:   "X-Frame-Options",
			expected: "DENY",
		},
		{
			name:     "Referrer-Policy",
			header:   "Referrer-Policy",
			expected: "strict-origin-when-cross-origin",
		},
		{
			name:   "Content-Security-Policy",
			header: "Content-Security-Policy",
			expected: "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://unpkg.com; " +
				"style-src 'self' 'unsafe-inline'; " +
				"img-src 'self' data:; " +
				"font-src 'self'; " +
				"connect-src 'self'; " +
				"frame-ancestors 'none'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := rec.Header().Get(tt.header)
			if actual != tt.expected {
				t.Errorf("expected %s header %q, got %q", tt.header, tt.expected, actual)
			}
		})
	}
}

func TestSecurityHeaders_AppliedToAllMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Check that security headers are set regardless of HTTP method
			if rec.Header().Get("X-Content-Type-Options") == "" {
				t.Errorf("expected X-Content-Type-Options header for %s method", method)
			}
		})
	}
}

func TestSecurityHeaders_NextHandlerCalled(t *testing.T) {
	handlerCalled := false
	handler := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected next handler to be called")
	}
}
