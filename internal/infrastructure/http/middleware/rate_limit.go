package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	requestsPerMinute int
	window            time.Duration
	visitors          map[string]*visitor
	mu                sync.Mutex
	lastCleanup       time.Time
	stop              chan struct{}
	stopOnce          sync.Once
}

// RateLimiterMiddleware is a per-client (IP-based) rate limiting middleware.
// Call Shutdown when the owning server is shutting down to stop background cleanup.
type RateLimiterMiddleware struct {
	rl *rateLimiter
}

func NewRateLimiterMiddleware(requestsPerMinute int, window time.Duration) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{rl: newRateLimiter(requestsPerMinute, window)}
}

// Middleware returns a chi-compatible middleware function.
func (m *RateLimiterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := m.rl.getLimiter(clientIP(r))

		if limiter.Allow() {
			next.ServeHTTP(w, r)
			return
		}

		retryAfter := m.rl.retryAfter(limiter)
		// Retry-After should be in whole seconds
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))

		respondRateLimitExceeded(w, r)
	})
}

func (m *RateLimiterMiddleware) Shutdown() {
	m.rl.shutdown()
}

func respondRateLimitExceeded(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") && !strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("<html><head><title>Too Many Requests</title></head><body><h1>Too Many Requests</h1><p>Rate limit exceeded. Please try again later.</p></body></html>"))
		return
	}

	respondJSONError(w, "Too Many Requests: rate limit exceeded", http.StatusTooManyRequests)
}

func newRateLimiter(requestsPerMinute int, window time.Duration) *rateLimiter {
	if requestsPerMinute < 1 {
		requestsPerMinute = 1
	}

	rl := &rateLimiter{
		requestsPerMinute: requestsPerMinute,
		window:            window,
		visitors:          make(map[string]*visitor),
		lastCleanup:       time.Now(),
		stop:              make(chan struct{}),
	}

	cleanupInterval := window
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}

	// Best-effort background cleanup to prevent unbounded growth if traffic patterns
	// don't trigger maybeCleanup frequently enough.
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case now := <-ticker.C:
				rl.cleanup(now)
			case <-rl.stop:
				return
			}
		}
	}()

	return rl
}

func (rl *rateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if v, ok := rl.visitors[key]; ok {
		v.lastSeen = now
		rl.maybeCleanup(now)
		return v.limiter
	}

	limit := rate.Every(rl.window / time.Duration(rl.requestsPerMinute))
	v := &visitor{
		limiter:  rate.NewLimiter(limit, rl.requestsPerMinute),
		lastSeen: now,
	}
	rl.visitors[key] = v
	rl.maybeCleanup(now)

	return v.limiter
}

func (rl *rateLimiter) retryAfter(limiter *rate.Limiter) time.Duration {
	reservation := limiter.Reserve()
	if !reservation.OK() {
		return rl.window
	}

	delay := reservation.Delay()
	reservation.Cancel()

	if delay < time.Second {
		return time.Second
	}

	return delay
}

func (rl *rateLimiter) maybeCleanup(now time.Time) {
	// Avoid frequent cleanup on hot paths
	if now.Sub(rl.lastCleanup) < rl.window {
		return
	}

	rl.cleanupLocked(now)
}

func (rl *rateLimiter) cleanup(now time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cleanupLocked(now)
}

func (rl *rateLimiter) cleanupLocked(now time.Time) {
	cutoff := now.Add(-rl.window)
	for key, v := range rl.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(rl.visitors, key)
		}
	}

	rl.lastCleanup = now
}

func (rl *rateLimiter) shutdown() {
	rl.stopOnce.Do(func() {
		close(rl.stop)
	})
}

func clientIP(r *http.Request) string {
	// NOTE: X-Forwarded-For and X-Real-IP can be spoofed by clients.
	// Only trust these headers if they are set/overwritten by a trusted reverse proxy.
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for _, p := range parts {
			if ip := strings.TrimSpace(p); ip != "" {
				return ip
			}
		}
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
