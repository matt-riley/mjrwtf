package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	ratelimiter := NewRateLimiterMiddleware(2, time.Minute)
	defer ratelimiter.Shutdown()

	calls := 0
	handler := ratelimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	ratelimiter := NewRateLimiterMiddleware(1, time.Minute)
	defer ratelimiter.Shutdown()

	handler := ratelimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	ratelimiter := NewRateLimiterMiddleware(1, time.Minute)
	defer ratelimiter.Shutdown()

	handler := ratelimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/xff", nil)
	req1.RemoteAddr = "192.0.2.100:1234"
	req1.Header.Set("X-Forwarded-For", "203.0.113.1")

	req2 := httptest.NewRequest(http.MethodGet, "/xff", nil)
	req2.RemoteAddr = "192.0.2.101:1234"
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

func TestClientIP_ForwardedForSkipsEmptySegments(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.50:1234"
	req.Header.Set("X-Forwarded-For", " , 203.0.113.9")

	ip := clientIP(req)
	if ip != "203.0.113.9" {
		t.Fatalf("expected IP %q, got %q", "203.0.113.9", ip)
	}
}

func TestRateLimiter_Cleanup_RemovesStaleVisitors(t *testing.T) {
	rl := newRateLimiter(1, 100*time.Millisecond)
	defer rl.shutdown()

	_ = rl.getLimiter("203.0.113.10")

	rl.mu.Lock()
	rl.visitors["203.0.113.10"].lastSeen = time.Now().Add(-time.Second)
	rl.mu.Unlock()

	rl.cleanup(time.Now())

	rl.mu.Lock()
	_, ok := rl.visitors["203.0.113.10"]
	rl.mu.Unlock()

	if ok {
		t.Fatal("expected stale visitor to be cleaned up")
	}
}

func TestClientIP_UsesRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.50:1234"
	req.Header.Set("X-Real-IP", "203.0.113.55")

	ip := clientIP(req)
	if ip != "203.0.113.55" {
		t.Fatalf("expected IP %q, got %q", "203.0.113.55", ip)
	}
}

func TestClientIP_RemoteAddrWithoutPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.77"

	ip := clientIP(req)
	if ip != "192.0.2.77" {
		t.Fatalf("expected IP %q, got %q", "192.0.2.77", ip)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	ratelimiter := NewRateLimiterMiddleware(1000, time.Minute)
	defer ratelimiter.Shutdown()

	handler := ratelimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)

	errCh := make(chan int, n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/concurrent", nil)
			req.RemoteAddr = "192.0.2.88:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				errCh <- rec.Code
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for code := range errCh {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
}

func TestRateLimiter_HTMLResponseWhenAcceptHTML(t *testing.T) {
	ratelimiter := NewRateLimiterMiddleware(1, time.Minute)
	defer ratelimiter.Shutdown()

	handler := ratelimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.0.2.99:1234"
	req1.Header.Set("Accept", "text/html")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = req1.RemoteAddr
	req2.Header.Set("Accept", "text/html")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, rec2.Code)
	}

	ct := rec2.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected Content-Type text/html, got %q", ct)
	}
}
