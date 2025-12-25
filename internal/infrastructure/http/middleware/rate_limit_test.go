package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestTokenBucket_Allow(t *testing.T) {
	// Create a bucket with 2 RPS and burst of 5
	tb := newTokenBucket(2, 5)

	// Should allow up to burst size immediately
	for i := 0; i < 5; i++ {
		allowed, _ := tb.allow()
		if !allowed {
			t.Errorf("request %d should be allowed (within burst)", i+1)
		}
	}

	// Next request should be denied (bucket empty)
	allowed, retryAfter := tb.allow()
	if allowed {
		t.Error("request should be denied after exhausting burst")
	}
	if retryAfter <= 0 {
		t.Errorf("retry after should be positive, got %v", retryAfter)
	}

	// Wait for refill
	time.Sleep(600 * time.Millisecond)

	// Should allow one more request (refilled ~1.2 tokens)
	allowed, _ = tb.allow()
	if !allowed {
		t.Error("request should be allowed after refill")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	// Create a bucket with 10 RPS and burst of 1
	tb := newTokenBucket(10, 1)

	// Use the one token
	allowed, _ := tb.allow()
	if !allowed {
		t.Error("first request should be allowed")
	}

	// Should be denied immediately
	allowed, _ = tb.allow()
	if allowed {
		t.Error("second request should be denied")
	}

	// Wait for 100ms (should refill 1 token at 10 RPS)
	time.Sleep(110 * time.Millisecond)

	// Should be allowed again
	allowed, _ = tb.allow()
	if !allowed {
		t.Error("request should be allowed after refill")
	}
}

func TestRateLimiter_PerClient(t *testing.T) {
	logger := zerolog.Nop()
	rl := newRateLimiter(2, 2, logger)

	// Client 1 should get 2 requests
	allowed, _ := rl.allow("client1")
	if !allowed {
		t.Error("client1 request 1 should be allowed")
	}
	allowed, _ = rl.allow("client1")
	if !allowed {
		t.Error("client1 request 2 should be allowed")
	}
	allowed, _ = rl.allow("client1")
	if allowed {
		t.Error("client1 request 3 should be denied")
	}

	// Client 2 should still get 2 requests (separate bucket)
	allowed, _ = rl.allow("client2")
	if !allowed {
		t.Error("client2 request 1 should be allowed")
	}
	allowed, _ = rl.allow("client2")
	if !allowed {
		t.Error("client2 request 2 should be allowed")
	}
	allowed, _ = rl.allow("client2")
	if allowed {
		t.Error("client2 request 3 should be denied")
	}
}

func TestRateLimitMiddleware_Allow(t *testing.T) {
	logger := zerolog.Nop()
	handlerCalled := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := IPBasedRateLimit(10, 2, logger)
	wrappedHandler := middleware(handler)

	// First request should succeed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("handler should have been called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRateLimitMiddleware_Deny(t *testing.T) {
	logger := zerolog.Nop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Very restrictive limit: 1 RPS, burst of 1
	middleware := IPBasedRateLimit(1, 1, logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request should succeed
	rec1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Errorf("first request: expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Second immediate request should be denied
	rec2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected status %d, got %d", http.StatusTooManyRequests, rec2.Code)
	}

	// Should have Retry-After header
	retryAfter := rec2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header should be present")
	}

	// Parse and validate Retry-After value
	seconds, err := strconv.ParseFloat(retryAfter, 64)
	if err != nil {
		t.Errorf("Retry-After should be a valid number: %v", err)
	}
	if seconds <= 0 {
		t.Errorf("Retry-After should be positive, got %v", seconds)
	}

	// Check response body for error message
	if rec2.Body.Len() == 0 {
		t.Error("response body should contain error message")
	}
}

func TestRateLimitMiddleware_DifferentIPs(t *testing.T) {
	logger := zerolog.Nop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Restrictive limit: 1 RPS, burst of 1
	middleware := IPBasedRateLimit(1, 1, logger)
	wrappedHandler := middleware(handler)

	// Request from IP 1
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("IP1 request: expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Immediate request from IP 2 should succeed (different bucket)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rec2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("IP2 request: expected status %d, got %d", http.StatusOK, rec2.Code)
	}

	// Second request from IP 1 should be denied
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	rec3 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 second request: expected status %d, got %d", http.StatusTooManyRequests, rec3.Code)
	}
}

func TestRateLimitMiddleware_XForwardedFor(t *testing.T) {
	logger := zerolog.Nop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := IPBasedRateLimit(1, 1, logger)
	wrappedHandler := middleware(handler)

	// First request with X-Forwarded-For
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("first request: expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Second request with same X-Forwarded-For should be denied
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req2.RemoteAddr = "192.168.1.1:12345"
	rec2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected status %d, got %d", http.StatusTooManyRequests, rec2.Code)
	}
}

func TestRateLimitMiddleware_XRealIP(t *testing.T) {
	logger := zerolog.Nop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := IPBasedRateLimit(1, 1, logger)
	wrappedHandler := middleware(handler)

	// First request with X-Real-IP
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("X-Real-IP", "10.0.0.5")
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("first request: expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Second request with same X-Real-IP should be denied
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Real-IP", "10.0.0.5")
	req2.RemoteAddr = "192.168.1.1:12345"
	rec2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected status %d, got %d", http.StatusTooManyRequests, rec2.Code)
	}
}

func TestGlobalRateLimit(t *testing.T) {
	logger := zerolog.Nop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Global limit: 2 RPS, burst of 2
	middleware := GlobalRateLimit(2, 2, logger)
	wrappedHandler := middleware(handler)

	// First two requests from different IPs should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("request 1: expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rec2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("request 2: expected status %d, got %d", http.StatusOK, rec2.Code)
	}

	// Third request from yet another IP should be denied (global limit)
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.3:12345"
	rec3 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("request 3: expected status %d, got %d", http.StatusTooManyRequests, rec3.Code)
	}
}

func TestRateLimitMiddleware_Concurrent(t *testing.T) {
	logger := zerolog.Nop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Allow 50 requests with burst of 50
	middleware := IPBasedRateLimit(100, 50, logger)
	wrappedHandler := middleware(handler)

	// Send 100 concurrent requests from same IP
	var wg sync.WaitGroup
	successCount := 0
	deniedCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			mu.Lock()
			if rec.Code == http.StatusOK {
				successCount++
			} else if rec.Code == http.StatusTooManyRequests {
				deniedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Should allow approximately burst size (50) and deny the rest
	// Allow some tolerance due to concurrent access and timing
	if successCount < 45 || successCount > 55 {
		t.Errorf("expected ~50 successes, got %d", successCount)
	}
	if deniedCount < 45 || deniedCount > 55 {
		t.Errorf("expected ~50 denials, got %d", deniedCount)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For single",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1",
			expectedIP:    "10.0.0.1",
		},
		{
			name:          "X-Forwarded-For multiple",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1, 10.0.0.2, 10.0.0.3",
			expectedIP:    "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:12345",
			xRealIP:    "10.0.0.5",
			expectedIP: "10.0.0.5",
		},
		{
			name:          "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1",
			xRealIP:       "10.0.0.5",
			expectedIP:    "10.0.0.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP %q, got %q", tt.expectedIP, ip)
			}
		})
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	logger := zerolog.Nop()
	rl := newRateLimiter(10, 10, logger)
	
	// Set short cleanup age for testing
	rl.cleanupAge = 100 * time.Millisecond

	// Create some buckets
	rl.allow("client1")
	rl.allow("client2")
	rl.allow("client3")

	if len(rl.buckets) != 3 {
		t.Errorf("expected 3 buckets, got %d", len(rl.buckets))
	}

	// Wait for cleanup age
	time.Sleep(150 * time.Millisecond)

	// Trigger cleanup by making a request
	rl.allow("client4")

	// Old buckets should be cleaned up
	if len(rl.buckets) != 1 {
		t.Errorf("expected 1 bucket after cleanup, got %d", len(rl.buckets))
	}
}
