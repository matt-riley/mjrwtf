package middleware

import (
	"context"
	"net/http"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/session"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "mjrwtf_session"
	
	// SessionUserIDKey is the context key for storing session user ID
	SessionUserIDKey contextKey = "sessionUserID"
)

// SessionMiddleware creates a middleware that manages user sessions
func SessionMiddleware(store *session.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get session cookie
			cookie, err := r.Cookie(SessionCookieName)
			if err == nil && cookie.Value != "" {
				// Validate session
				sess, exists := store.Get(cookie.Value)
				if exists {
					// Refresh session on each request
					store.Refresh(cookie.Value)
					
					// Add user ID to context
					ctx := context.WithValue(r.Context(), SessionUserIDKey, sess.UserID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			
			// No valid session, continue without user context
			next.ServeHTTP(w, r)
		})
	}
}

// RequireSession creates a middleware that requires a valid session
// Redirects to login page if session is not found
func RequireSession(store *session.Store, loginURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if user ID is in context (set by SessionMiddleware)
			userID, ok := GetSessionUserID(r.Context())
			if !ok || userID == "" {
				// No valid session, redirect to login
				http.Redirect(w, r, loginURL, http.StatusSeeOther)
				return
			}
			
			// Valid session, continue
			next.ServeHTTP(w, r)
		})
	}
}

// GetSessionUserID extracts the session user ID from the request context
func GetSessionUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(SessionUserIDKey).(string)
	return userID, ok
}

// SetSessionCookie sets the session cookie with secure defaults
func SetSessionCookie(w http.ResponseWriter, sessionID string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
