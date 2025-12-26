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
}

// RateLimit returns a middleware that enforces a per-client rate limit.
// Limits are tracked per remote IP (preferring X-Forwarded-For when present).
func RateLimit(requestsPerMinute int, window time.Duration) func(http.Handler) http.Handler {
	rl := newRateLimiter(requestsPerMinute, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter := rl.getLimiter(clientIP(r))

			if limiter.Allow() {
				next.ServeHTTP(w, r)
				return
			}

			retryAfter := rl.retryAfter(limiter)
			// Retry-After should be in whole seconds
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))

			respondJSONError(w, "Too Many Requests: rate limit exceeded", http.StatusTooManyRequests)
		})
	}
}

func newRateLimiter(requestsPerMinute int, window time.Duration) *rateLimiter {
	if requestsPerMinute < 1 {
		requestsPerMinute = 1
	}

	return &rateLimiter{
		requestsPerMinute: requestsPerMinute,
		window:            window,
		visitors:          make(map[string]*visitor),
		lastCleanup:       time.Now(),
	}
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

	cutoff := now.Add(-rl.window)
	for key, v := range rl.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(rl.visitors, key)
		}
	}

	rl.lastCleanup = now
}

func clientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
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
