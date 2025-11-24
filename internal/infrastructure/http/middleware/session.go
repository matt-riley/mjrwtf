package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/session"
)

const (
	// SessionKey is the context key for storing session data
	SessionKey contextKey = "session"

	// SessionCookieName is the name of the session cookie
	SessionCookieName = "session_id"

	// DefaultSessionTimeout is the default idle timeout (30 minutes)
	DefaultSessionTimeout = 30 * time.Minute

	// SessionCookieMaxAge is the maximum age for session cookies (24 hours)
	SessionCookieMaxAge = 86400 // 24 hours in seconds
)

// SessionMiddleware handles session-based authentication
type SessionMiddleware struct {
	sessionRepo    session.Repository
	sessionTimeout time.Duration
	secureCookies  bool
}

// NewSessionMiddleware creates a new session middleware
func NewSessionMiddleware(sessionRepo session.Repository, sessionTimeout time.Duration, secureCookies bool) *SessionMiddleware {
	if sessionTimeout == 0 {
		sessionTimeout = DefaultSessionTimeout
	}

	return &SessionMiddleware{
		sessionRepo:    sessionRepo,
		sessionTimeout: sessionTimeout,
		secureCookies:  secureCookies,
	}
}

// RequireSession returns middleware that requires a valid session
func (sm *SessionMiddleware) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session ID from cookie
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			respondJSONError(w, "Unauthorized: missing session", http.StatusUnauthorized)
			return
		}

		sessionID := cookie.Value
		if sessionID == "" {
			respondJSONError(w, "Unauthorized: invalid session cookie", http.StatusUnauthorized)
			return
		}

		// Validate session ID format
		if err := session.ValidateSessionID(sessionID); err != nil {
			respondJSONError(w, "Unauthorized: invalid session ID format", http.StatusUnauthorized)
			return
		}

		// Load session from repository
		sess, err := sm.sessionRepo.FindByID(r.Context(), sessionID)
		if err != nil {
			if err == session.ErrSessionNotFound {
				respondJSONError(w, "Unauthorized: session not found", http.StatusUnauthorized)
				return
			}
			respondJSONError(w, "Unauthorized: failed to load session", http.StatusUnauthorized)
			return
		}

		// Check if session expired
		if sess.IsExpired() {
			respondJSONError(w, "Unauthorized: session expired", http.StatusUnauthorized)
			return
		}

		// Check if session idle
		if sess.IsIdle(sm.sessionTimeout) {
			respondJSONError(w, "Unauthorized: session idle timeout", http.StatusUnauthorized)
			return
		}

		// Update activity timestamp
		sess.UpdateActivity()
		if err := sm.sessionRepo.UpdateActivity(r.Context(), sess.ID, sess.LastActivityAt); err != nil {
			// Log error but continue - this is not critical
			// In production, you might want to log this for monitoring
		}

		// Store session and user ID in context
		ctx := context.WithValue(r.Context(), SessionKey, sess)
		ctx = context.WithValue(ctx, UserIDKey, sess.UserID)

		// Continue with authenticated request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalSession returns middleware that loads session if present but continues if not
func (sm *SessionMiddleware) OptionalSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to extract session ID from cookie
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			// No session cookie - continue without session
			next.ServeHTTP(w, r)
			return
		}

		sessionID := cookie.Value
		if sessionID == "" {
			// Empty session cookie - continue without session
			next.ServeHTTP(w, r)
			return
		}

		// Validate session ID format
		if err := session.ValidateSessionID(sessionID); err != nil {
			// Invalid format - continue without session
			next.ServeHTTP(w, r)
			return
		}

		// Try to load session from repository
		sess, err := sm.sessionRepo.FindByID(r.Context(), sessionID)
		if err != nil {
			// Session not found or error - continue without session
			next.ServeHTTP(w, r)
			return
		}

		// Check if session is valid
		if sess.IsExpired() || sess.IsIdle(sm.sessionTimeout) {
			// Expired or idle - continue without session
			next.ServeHTTP(w, r)
			return
		}

		// Update activity timestamp
		sess.UpdateActivity()
		if err := sm.sessionRepo.UpdateActivity(r.Context(), sess.ID, sess.LastActivityAt); err != nil {
			// Log error but continue
		}

		// Store session and user ID in context
		ctx := context.WithValue(r.Context(), SessionKey, sess)
		ctx = context.WithValue(ctx, UserIDKey, sess.UserID)

		// Continue with session in context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetSession retrieves the session from the request context
func GetSession(ctx context.Context) (*session.Session, bool) {
	sess, ok := ctx.Value(SessionKey).(*session.Session)
	return sess, ok
}

// SetSessionCookie sets the session cookie on the response
func SetSessionCookie(w http.ResponseWriter, sessionID string, maxAge int, secureCookies bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie clears the session cookie (for logout)
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
