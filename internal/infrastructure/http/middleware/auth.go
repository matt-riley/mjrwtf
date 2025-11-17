package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for storing user identity
	UserIDKey contextKey = "userID"
)

// Auth returns a middleware that validates Bearer token authentication
func Auth(authToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized: missing authorization header", http.StatusUnauthorized)
				return
			}

			// Check for Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Unauthorized: invalid authorization format", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token against configured secret
			if token != authToken {
				http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
				return
			}

			// Extract user identity (for now, use a static identifier)
			// In future iterations, this could be extracted from JWT claims
			userID := "authenticated-user"

			// Add user identity to request context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)

			// Continue with authenticated request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
