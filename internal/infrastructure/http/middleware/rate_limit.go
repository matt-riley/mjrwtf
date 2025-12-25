package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// tokenBucket implements a token bucket rate limiter
type tokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	mu             sync.Mutex
}

// newTokenBucket creates a new token bucket with the specified rate and burst
func newTokenBucket(rps int, burst int) *tokenBucket {
	now := time.Now()
	return &tokenBucket{
		tokens:         float64(burst),
		maxTokens:      float64(burst),
		refillRate:     float64(rps),
		lastRefillTime: now,
	}
}

// allow checks if a request can proceed and consumes a token if so
// Returns true if allowed, false otherwise, and the time until the next token
func (tb *tokenBucket) allow() (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()

	// Refill tokens based on elapsed time
	tb.tokens = min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefillTime = now

	// Check if we have enough tokens
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true, 0
	}

	// Calculate time until next token is available
	tokensNeeded := 1.0 - tb.tokens
	waitTime := time.Duration(tokensNeeded/tb.refillRate*1000) * time.Millisecond
	return false, waitTime
}

// rateLimiter manages rate limiting for multiple clients
type rateLimiter struct {
	buckets    map[string]*tokenBucket
	mu         sync.RWMutex
	rps        int
	burst      int
	cleanupAge time.Duration
	lastCleanup time.Time
	logger     zerolog.Logger
}

// newRateLimiter creates a new rate limiter with the specified rate and burst
func newRateLimiter(rps int, burst int, logger zerolog.Logger) *rateLimiter {
	return &rateLimiter{
		buckets:     make(map[string]*tokenBucket),
		rps:         rps,
		burst:       burst,
		cleanupAge:  10 * time.Minute,
		lastCleanup: time.Now(),
		logger:      logger,
	}
}

// allow checks if a request from the given client is allowed
func (rl *rateLimiter) allow(clientID string) (bool, time.Duration) {
	// Periodic cleanup to prevent memory leaks
	rl.periodicCleanup()

	rl.mu.RLock()
	bucket, exists := rl.buckets[clientID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		bucket, exists = rl.buckets[clientID]
		if !exists {
			bucket = newTokenBucket(rl.rps, rl.burst)
			rl.buckets[clientID] = bucket
		}
		rl.mu.Unlock()
	}

	return bucket.allow()
}

// periodicCleanup removes old token buckets to prevent memory leaks
func (rl *rateLimiter) periodicCleanup() {
	rl.mu.RLock()
	needsCleanup := time.Since(rl.lastCleanup) > rl.cleanupAge
	rl.mu.RUnlock()

	if !needsCleanup {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(rl.lastCleanup) <= rl.cleanupAge {
		return
	}

	now := time.Now()
	beforeCount := len(rl.buckets)
	
	// Remove buckets that haven't been used recently
	for clientID, bucket := range rl.buckets {
		bucket.mu.Lock()
		lastUsed := bucket.lastRefillTime
		bucket.mu.Unlock()

		if now.Sub(lastUsed) > rl.cleanupAge {
			delete(rl.buckets, clientID)
		}
	}

	afterCount := len(rl.buckets)
	if beforeCount > afterCount {
		rl.logger.Debug().
			Int("before", beforeCount).
			Int("after", afterCount).
			Int("removed", beforeCount-afterCount).
			Msg("rate limiter cleanup completed")
	}

	rl.lastCleanup = now
}

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	RPS        int
	Burst      int
	KeyFunc    func(*http.Request) string
	Logger     zerolog.Logger
}

// RateLimit returns a middleware that applies rate limiting based on the provided configuration
func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	limiter := newRateLimiter(cfg.RPS, cfg.Burst, cfg.Logger)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientID := cfg.KeyFunc(r)
			
			allowed, retryAfter := limiter.allow(clientID)
			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				respondJSONError(w, "Rate limit exceeded", http.StatusTooManyRequests)
				
				cfg.Logger.Warn().
					Str("client_id", clientID).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Dur("retry_after", retryAfter).
					Msg("rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPBasedRateLimit creates a rate limiting middleware that limits by IP address
func IPBasedRateLimit(rps int, burst int, logger zerolog.Logger) func(http.Handler) http.Handler {
	return RateLimit(RateLimitConfig{
		RPS:     rps,
		Burst:   burst,
		KeyFunc: getClientIP,
		Logger:  logger,
	})
}

// GlobalRateLimit creates a rate limiting middleware that applies a global limit
func GlobalRateLimit(rps int, burst int, logger zerolog.Logger) func(http.Handler) http.Handler {
	return RateLimit(RateLimitConfig{
		RPS:     rps,
		Burst:   burst,
		KeyFunc: func(r *http.Request) string { return "global" },
		Logger:  logger,
	})
}

// getClientIP extracts the client IP from the request
// It checks X-Forwarded-For and X-Real-IP headers first (for reverse proxy scenarios)
// then falls back to RemoteAddr
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (comma-separated list, first is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
