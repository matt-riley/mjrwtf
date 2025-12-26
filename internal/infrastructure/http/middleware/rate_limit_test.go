package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	middleware := RateLimit(2, time.Minute)

	calls := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	}

	if calls != 2 {
		t.Fatalf("expected handler to be called twice, got %d", calls)
	}
}

func TestRateLimit_BlocksWhenExceeded(t *testing.T) {
	middleware := RateLimit(1, time.Minute)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.10:5050"

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, second.Code)
	}

	retryAfter := second.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header to be set")
	}

	retrySeconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("expected Retry-After to be an integer, got %s", retryAfter)
	}

	if retrySeconds < 1 {
		t.Fatalf("expected Retry-After to be at least 1 second, got %d", retrySeconds)
	}
}

func TestRateLimit_UsesForwardedFor(t *testing.T) {
	middleware := RateLimit(1, time.Minute)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/xff", nil)
	req1.Header.Set("X-Forwarded-For", "203.0.113.1")

	req2 := httptest.NewRequest(http.MethodGet, "/xff", nil)
	req2.Header.Set("X-Forwarded-For", "203.0.113.2")

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status %d for first IP, got %d", http.StatusOK, rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected different IP to have separate limit, got %d", rec2.Code)
	}
}
