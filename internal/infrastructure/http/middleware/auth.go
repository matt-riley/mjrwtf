package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for storing user identity
	UserIDKey contextKey = "userID"
)

// Auth returns a middleware that validates Bearer token authentication.
//
// It accepts multiple active tokens to support zero-downtime rotations.
func Auth(authTokens []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondJSONError(w, "Unauthorized: missing authorization header", http.StatusUnauthorized)
				return
			}

			// Check for Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondJSONError(w, "Unauthorized: invalid authorization format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			if token == "" {
				respondJSONError(w, "Unauthorized: invalid authorization format", http.StatusUnauthorized)
				return
			}

			match, configured := ValidateTokenConstantTime(token, authTokens)
			if !configured {
				respondJSONError(w, "Unauthorized: no valid tokens configured", http.StatusUnauthorized)
				return
			}
			if !match {
				respondJSONError(w, "Unauthorized: invalid token", http.StatusUnauthorized)
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

// respondJSONError writes a JSON error response
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Use json.Marshal to properly escape the message
	type errorResponse struct {
		Error string `json:"error"`
	}

	response := errorResponse{Error: message}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback to plain text if JSON encoding fails
		w.Write([]byte(`{"error":"internal server error"}`))
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
