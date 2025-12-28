package middleware

import (
	"context"
	"net/http"
)

// SessionOrBearerAuth authorizes a request if either:
//   - a valid session user is present in context (from SessionMiddleware), or
//   - a valid Bearer token is provided (via Auth).
//
// If a session user is present, it is preferred over Bearer auth.
func SessionOrBearerAuth(authTokens []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, ok := GetSessionUserID(r.Context()); ok && userID != "" {
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			Auth(authTokens)(next).ServeHTTP(w, r)
		})
	}
}
