# Secure Session Management Design for mjr.wtf

## Executive Summary

This document provides a comprehensive design for replacing the insecure sessionStorage-based authentication with secure server-side session management using httpOnly cookies. The design follows OWASP security best practices and maintains the hexagonal architecture pattern used throughout the application.

## Current State Analysis

### Security Issues with Current Implementation

**Critical Vulnerabilities:**
1. **sessionStorage exposure**: Auth tokens stored in sessionStorage are accessible to JavaScript, making them vulnerable to XSS attacks
2. **No session lifecycle**: Tokens don't expire, creating unlimited attack windows
3. **No session invalidation**: Users cannot log out or revoke sessions
4. **No CSRF protection**: Form-based authentication lacks CSRF token validation
5. **Single static user**: System uses hardcoded "authenticated-user" ID
6. **Token reuse**: Same AUTH_TOKEN shared across all authenticated users

**Current Architecture:**
- Bearer token authentication for API endpoints (`/api/*`)
- Form-based authentication for HTML pages (`/create`, `/dashboard`)
- Auth token stored in frontend sessionStorage
- Middleware validates token using constant-time comparison
- Single static user ID for all authenticated requests

## Proposed Solution: Secure Session Management

### 1. Session Data Structure

#### Domain Entity: `internal/domain/session/session.go`

```go
package session

import (
    "crypto/rand"
    "encoding/base64"
    "time"
)

// Session represents an authenticated user session
type Session struct {
    ID           string    // Cryptographically secure random ID
    UserID       string    // User identifier (future: link to users table)
    CreatedAt    time.Time // When session was created
    ExpiresAt    time.Time // When session expires (absolute)
    LastActivity time.Time // Last request timestamp (for idle timeout)
    IPAddress    string    // Client IP for security monitoring
    UserAgent    string    // Client UA for security monitoring
}

// SessionConfig holds session configuration
type SessionConfig struct {
    MaxAge         time.Duration // Maximum session lifetime (e.g., 24h)
    IdleTimeout    time.Duration // Idle timeout (e.g., 30m)
    SecureOnly     bool          // Require HTTPS in production
    SameSite       string        // "Strict", "Lax", or "None"
    CookieName     string        // Cookie name (e.g., "mjrwtf_session")
    CookiePath     string        // Cookie path (default: "/")
    CookieDomain   string        // Cookie domain (empty for same-origin)
}

// DefaultSessionConfig returns production-ready defaults
func DefaultSessionConfig() SessionConfig {
    return SessionConfig{
        MaxAge:       24 * time.Hour,
        IdleTimeout:  30 * time.Minute,
        SecureOnly:   true,
        SameSite:     "Lax",
        CookieName:   "mjrwtf_session",
        CookiePath:   "/",
        CookieDomain: "",
    }
}

// NewSession creates a new session with cryptographically secure ID
func NewSession(userID string, config SessionConfig, ipAddress, userAgent string) (*Session, error) {
    sessionID, err := generateSecureSessionID()
    if err != nil {
        return nil, err
    }

    now := time.Now()
    return &Session{
        ID:           sessionID,
        UserID:       userID,
        CreatedAt:    now,
        ExpiresAt:    now.Add(config.MaxAge),
        LastActivity: now,
        IPAddress:    ipAddress,
        UserAgent:    userAgent,
    }, nil
}

// IsExpired checks if session has expired (absolute or idle timeout)
func (s *Session) IsExpired(idleTimeout time.Duration) bool {
    now := time.Now()
    
    // Check absolute expiration
    if now.After(s.ExpiresAt) {
        return true
    }
    
    // Check idle timeout
    if idleTimeout > 0 && now.Sub(s.LastActivity) > idleTimeout {
        return true
    }
    
    return false
}

// UpdateActivity updates last activity timestamp
func (s *Session) UpdateActivity() {
    s.LastActivity = time.Now()
}

// generateSecureSessionID generates a cryptographically secure random session ID
// Returns 32 bytes (256 bits) encoded as base64 URL-safe string
func generateSecureSessionID() (string, error) {
    // 32 bytes = 256 bits of entropy
    // Base64 encoding increases size to 43 characters
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    
    // Use URL-safe base64 encoding without padding
    return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

// Validate validates session entity
func (s *Session) Validate() error {
    if s.ID == "" {
        return ErrInvalidSessionID
    }
    
    if len(s.ID) < 32 {
        return ErrSessionIDTooShort
    }
    
    if s.UserID == "" {
        return ErrInvalidUserID
    }
    
    if s.CreatedAt.IsZero() {
        return ErrInvalidCreatedAt
    }
    
    if s.ExpiresAt.IsZero() || s.ExpiresAt.Before(s.CreatedAt) {
        return ErrInvalidExpiresAt
    }
    
    return nil
}
```

#### Domain Errors: `internal/domain/session/errors.go`

```go
package session

import "errors"

var (
    // ErrSessionNotFound is returned when a session cannot be found
    ErrSessionNotFound = errors.New("session not found")
    
    // ErrSessionExpired is returned when a session has expired
    ErrSessionExpired = errors.New("session has expired")
    
    // ErrInvalidSessionID is returned when session ID is invalid
    ErrInvalidSessionID = errors.New("invalid session ID")
    
    // ErrSessionIDTooShort is returned when session ID is too short
    ErrSessionIDTooShort = errors.New("session ID must be at least 32 characters")
    
    // ErrInvalidUserID is returned when user ID is invalid
    ErrInvalidUserID = errors.New("invalid user ID")
    
    // ErrInvalidCreatedAt is returned when created_at is invalid
    ErrInvalidCreatedAt = errors.New("invalid created_at timestamp")
    
    // ErrInvalidExpiresAt is returned when expires_at is invalid
    ErrInvalidExpiresAt = errors.New("invalid expires_at timestamp")
    
    // ErrSecurityViolation is returned when session security check fails
    ErrSecurityViolation = errors.New("session security violation detected")
)
```

#### Repository Interface: `internal/domain/session/repository.go`

```go
package session

import "context"

// Repository defines operations for session persistence
type Repository interface {
    // Create stores a new session
    Create(ctx context.Context, session *Session) error
    
    // FindByID retrieves a session by ID
    FindByID(ctx context.Context, id string) (*Session, error)
    
    // UpdateActivity updates the last activity timestamp
    UpdateActivity(ctx context.Context, id string, lastActivity time.Time) error
    
    // Delete removes a session (logout)
    Delete(ctx context.Context, id string) error
    
    // DeleteByUserID removes all sessions for a user
    DeleteByUserID(ctx context.Context, userID string) error
    
    // DeleteExpired removes all expired sessions (cleanup)
    DeleteExpired(ctx context.Context) (int64, error)
    
    // ListByUserID retrieves all active sessions for a user
    ListByUserID(ctx context.Context, userID string) ([]*Session, error)
}
```

### 2. Database Schema

#### SQLite Migration: `internal/migrations/sqlite/00002_sessions.sql`

```sql
-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Sessions Table
-- ============================================================================
-- Stores server-side session data for authenticated users
--
-- Security considerations:
-- - session_id is cryptographically secure (256 bits)
-- - Expires_at enforces absolute session lifetime
-- - last_activity enables idle timeout detection
-- - ip_address and user_agent for security monitoring
CREATE TABLE IF NOT EXISTS sessions (
    -- Session ID: cryptographically secure random identifier
    -- Base64-encoded 32-byte (256-bit) random value
    -- Example: "xyz123..." (43 characters)
    id VARCHAR(255) PRIMARY KEY NOT NULL,
    
    -- User identifier who owns this session
    -- Future: foreign key to users table when user management is implemented
    user_id VARCHAR(255) NOT NULL,
    
    -- Timestamp when session was created
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Absolute expiration timestamp
    -- Session is invalid after this time regardless of activity
    -- Typically: created_at + 24 hours
    expires_at TIMESTAMP NOT NULL,
    
    -- Last activity timestamp
    -- Updated on each authenticated request
    -- Used for idle timeout detection (e.g., 30 minutes)
    last_activity TIMESTAMP NOT NULL,
    
    -- Client IP address when session was created
    -- Used for security monitoring and anomaly detection
    -- IPv4 example: "192.168.1.1"
    -- IPv6 example: "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
    ip_address VARCHAR(45) NOT NULL,
    
    -- User-Agent string when session was created
    -- Used for device identification and security monitoring
    user_agent TEXT
);

-- Index for finding sessions by user (for listing user's active sessions)
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- Index for cleanup queries (finding expired sessions)
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Index for idle timeout queries
CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd
```

#### PostgreSQL Migration: `internal/migrations/postgres/00002_sessions.sql`

```sql
-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Sessions Table
-- ============================================================================
-- Stores server-side session data for authenticated users
CREATE TABLE IF NOT EXISTS sessions (
    -- Session ID: cryptographically secure random identifier
    id VARCHAR(255) PRIMARY KEY NOT NULL,
    
    -- User identifier who owns this session
    user_id VARCHAR(255) NOT NULL,
    
    -- Timestamp when session was created
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Absolute expiration timestamp
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Last activity timestamp
    last_activity TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Client IP address
    ip_address VARCHAR(45) NOT NULL,
    
    -- User-Agent string
    user_agent TEXT
);

-- Index for finding sessions by user
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- Index for cleanup queries
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Index for idle timeout queries
CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd
```

### 3. Session Lifecycle Management

#### Session Creation (Login)

**Flow:**
1. User submits login form with auth token (temporary approach)
2. Server validates auth token using constant-time comparison
3. Generate secure session ID (32 bytes, 256 bits)
4. Create session record in database
5. Set httpOnly cookie with session ID
6. Redirect to dashboard

**Security Considerations:**
- Use crypto/rand for session ID generation
- Validate token with subtle.ConstantTimeCompare to prevent timing attacks
- Set httpOnly flag to prevent JavaScript access
- Set Secure flag to require HTTPS in production
- Set SameSite=Lax to prevent CSRF on most requests
- Store minimal data in cookie (only session ID)
- Log session creation for security monitoring

#### Session Validation (Request Authentication)

**Flow:**
1. Extract session ID from cookie
2. Look up session in database
3. Check absolute expiration (expires_at)
4. Check idle timeout (last_activity)
5. Validate IP address (optional, can break with mobile networks)
6. Update last_activity timestamp
7. Add user ID to request context
8. Continue to handler

**Security Considerations:**
- Validate session on every authenticated request
- Use constant-time comparison for session ID lookups
- Update last_activity to extend idle timeout
- Detect session anomalies (IP changes, multiple concurrent locations)
- Rate limit session validation to prevent DoS
- Log validation failures for security monitoring

#### Session Renewal

**Strategy:** Sliding window with activity updates

**Implementation:**
- Update last_activity on each request
- Idle timeout: 30 minutes (configurable)
- Absolute timeout: 24 hours (configurable)
- No automatic renewal near expiration (user must re-authenticate)

**Security Rationale:**
- Limits damage from session theft
- Forces periodic re-authentication
- Detects abandoned sessions
- Prevents infinite session lifetime

#### Session Destruction (Logout)

**Flow:**
1. User clicks logout button
2. Delete session from database
3. Clear session cookie (set MaxAge=-1)
4. Redirect to home page

**Additional Logout Scenarios:**
- Logout all sessions (user-initiated security action)
- Force logout on password change (if implemented)
- Automatic cleanup of expired sessions (background job)

### 4. Cookie Configuration

#### Production Settings

```go
http.SetCookie(w, &http.Cookie{
    Name:     "mjrwtf_session",
    Value:    sessionID,
    Path:     "/",
    Domain:   "",              // Empty = same origin only
    MaxAge:   86400,           // 24 hours in seconds
    Secure:   true,            // HTTPS only
    HttpOnly: true,            // No JavaScript access
    SameSite: http.SameSiteLaxMode, // CSRF protection
})
```

#### Development Settings

```go
http.SetCookie(w, &http.Cookie{
    Name:     "mjrwtf_session",
    Value:    sessionID,
    Path:     "/",
    Domain:   "",
    MaxAge:   86400,
    Secure:   false,           // Allow HTTP in development
    HttpOnly: true,
    SameSite: http.SameSiteLaxMode,
})
```

#### Cookie Attributes Explained

**HttpOnly: true**
- Prevents JavaScript from accessing cookie
- Mitigates XSS attacks
- **Critical security control**

**Secure: true (production)**
- Cookie only sent over HTTPS
- Prevents interception on unsecured networks
- **Required for production**

**SameSite: Lax**
- Cookie sent with top-level navigations (GET)
- Cookie NOT sent with cross-site POST requests
- Prevents most CSRF attacks
- Allows legitimate external links to work

**SameSite Options:**
- **Strict**: Maximum security, breaks external links
- **Lax**: Balanced security, recommended for most sites
- **None**: No CSRF protection, requires Secure flag

**MaxAge: 86400**
- Cookie expires in 24 hours (client-side enforcement)
- Matches server-side session expiration
- Browsers delete cookie after expiration

**Path: /**
- Cookie sent for all paths on domain
- Matches application routing structure

**Domain: "" (empty)**
- Cookie restricted to exact origin
- Prevents subdomain access
- Tightest security boundary

### 5. Security Considerations

#### CSRF Protection

**Current Risk:**
- Form-based authentication vulnerable to CSRF
- Attacker can submit forms from malicious site

**Mitigation Strategy:**

1. **SameSite Cookie** (Primary defense)
   - Set SameSite=Lax on session cookie
   - Prevents cookie from being sent with cross-site POST
   - Works on modern browsers

2. **CSRF Token** (Defense in depth)
   - Generate random token per session
   - Store in session database
   - Embed in forms as hidden field
   - Validate on POST requests

**Implementation:** `internal/domain/session/csrf.go`

```go
package session

import (
    "crypto/rand"
    "encoding/base64"
)

// GenerateCSRFToken creates a cryptographically secure CSRF token
func GenerateCSRFToken() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateCSRFToken compares tokens using constant-time comparison
func ValidateCSRFToken(token1, token2 string) bool {
    return subtle.ConstantTimeCompare([]byte(token1), []byte(token2)) == 1
}
```

**HTML Form Template:**
```html
<form method="POST" action="/create">
    <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
    <input type="url" name="original_url" required>
    <button type="submit">Shorten URL</button>
</form>
```

**Middleware Validation:**
```go
func (h *PageHandler) validateCSRFToken(r *http.Request, sessionCSRFToken string) bool {
    formToken := r.FormValue("csrf_token")
    return session.ValidateCSRFToken(formToken, sessionCSRFToken)
}
```

#### Session Fixation Prevention

**Attack Vector:**
- Attacker obtains valid session ID
- Tricks victim into using attacker's session
- Victim authenticates
- Attacker gains access to victim's account

**Mitigation:**
1. **Generate new session ID on login**
   - Never reuse pre-authentication session IDs
   - Destroy old session if exists
   - Create fresh session after authentication

2. **Regenerate session ID on privilege escalation**
   - Future: when user role changes
   - Future: after password change

**Implementation:**
```go
// On successful authentication:
// 1. Delete any existing session for this user
sessionRepo.DeleteByUserID(ctx, userID)

// 2. Generate new session with fresh ID
session, err := session.NewSession(userID, config, ip, ua)

// 3. Store new session
sessionRepo.Create(ctx, session)

// 4. Set cookie with new session ID
setSessionCookie(w, session.ID)
```

#### Session Hijacking Prevention

**Attack Vectors:**
1. Network interception (MITM)
2. XSS attacks
3. Cookie theft via malware

**Mitigations:**

**1. HTTPS Enforcement (Production)**
```go
// Redirect HTTP to HTTPS in production
func (s *Server) enforceHTTPS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !s.config.IsProduction {
            next.ServeHTTP(w, r)
            return
        }
        
        if r.TLS == nil {
            target := "https://" + r.Host + r.URL.Path
            if r.URL.RawQuery != "" {
                target += "?" + r.URL.RawQuery
            }
            http.Redirect(w, r, target, http.StatusMovedPermanently)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

**2. Security Headers**
```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent MIME type sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")
        
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // Enable XSS protection
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // HSTS - Force HTTPS for 1 year
        w.Header().Set("Strict-Transport-Security", 
            "max-age=31536000; includeSubDomains")
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", 
            "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
        
        // Referrer policy
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next.ServeHTTP(w, r)
    })
}
```

**3. IP Address Validation (Optional)**
```go
// Detect session hijacking via IP change
func (s *Session) ValidateIPAddress(currentIP string, strict bool) error {
    if !strict {
        return nil // Don't enforce IP validation
    }
    
    // Allow same IP
    if s.IPAddress == currentIP {
        return nil
    }
    
    // Optional: Allow same /24 subnet for mobile networks
    // Implementation would parse IPs and compare subnets
    
    return ErrSecurityViolation
}
```

**Note:** IP validation can break legitimate use cases:
- Mobile networks (IP changes frequently)
- VPN switching
- Load balancers (different egress IPs)
- Consider logging IP changes instead of blocking

#### Timing Attack Prevention

**Vulnerable Code:**
```go
// DON'T DO THIS
if token == expectedToken {
    return true
}
```

**Secure Code:**
```go
// Always use constant-time comparison
if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1 {
    return true
}
```

**Applied to:**
- Auth token validation
- Session ID lookups
- CSRF token validation
- Any security-critical comparison

#### Session Cleanup

**Background Job:** Automatic cleanup of expired sessions

```go
// internal/infrastructure/jobs/session_cleanup.go
package jobs

import (
    "context"
    "log"
    "time"
    
    "github.com/matt-riley/mjrwtf/internal/domain/session"
)

// SessionCleanup runs periodic cleanup of expired sessions
type SessionCleanup struct {
    repo     session.Repository
    interval time.Duration
    done     chan struct{}
}

func NewSessionCleanup(repo session.Repository, interval time.Duration) *SessionCleanup {
    return &SessionCleanup{
        repo:     repo,
        interval: interval,
        done:     make(chan struct{}),
    }
}

func (c *SessionCleanup) Start() {
    ticker := time.NewTicker(c.interval)
    go func() {
        for {
            select {
            case <-ticker.C:
                ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
                count, err := c.repo.DeleteExpired(ctx)
                cancel()
                
                if err != nil {
                    log.Printf("Session cleanup failed: %v", err)
                } else if count > 0 {
                    log.Printf("Cleaned up %d expired sessions", count)
                }
            case <-c.done:
                ticker.Stop()
                return
            }
        }
    }()
}

func (c *SessionCleanup) Stop() {
    close(c.done)
}
```

**Configuration:**
- Run every 15 minutes
- Clean sessions where expires_at < NOW OR last_activity < NOW - idle_timeout

### 6. Integration with Existing Middleware

#### New Session Middleware: `internal/infrastructure/http/middleware/session.go`

```go
package middleware

import (
    "context"
    "log"
    "net/http"
    "time"
    
    "github.com/matt-riley/mjrwtf/internal/domain/session"
)

// contextKey is a custom type for context keys to avoid collisions
type sessionContextKey string

const (
    // SessionKey is the context key for storing session
    SessionKey sessionContextKey = "session"
)

// SessionMiddleware validates session cookies and manages session lifecycle
type SessionMiddleware struct {
    repo   session.Repository
    config session.SessionConfig
}

func NewSessionMiddleware(repo session.Repository, config session.SessionConfig) *SessionMiddleware {
    return &SessionMiddleware{
        repo:   repo,
        config: config,
    }
}

// RequireSession enforces authentication via session cookie
func (m *SessionMiddleware) RequireSession(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        sess, err := m.validateSession(r)
        if err != nil {
            // Session invalid or expired
            m.clearSessionCookie(w)
            
            // Redirect to login for page requests, 401 for API
            if isAPIRequest(r) {
                respondJSONError(w, "Unauthorized: invalid or expired session", 
                    http.StatusUnauthorized)
            } else {
                http.Redirect(w, r, "/?error=session_expired", 
                    http.StatusSeeOther)
            }
            return
        }
        
        // Update last activity
        sess.UpdateActivity()
        if err := m.repo.UpdateActivity(r.Context(), sess.ID, sess.LastActivity); err != nil {
            log.Printf("Failed to update session activity: %v", err)
            // Continue anyway - don't fail request
        }
        
        // Add session to request context
        ctx := context.WithValue(r.Context(), SessionKey, sess)
        ctx = context.WithValue(ctx, UserIDKey, sess.UserID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// OptionalSession attempts to load session but doesn't require it
func (m *SessionMiddleware) OptionalSession(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        sess, err := m.validateSession(r)
        if err == nil && sess != nil {
            // Update last activity
            sess.UpdateActivity()
            m.repo.UpdateActivity(r.Context(), sess.ID, sess.LastActivity)
            
            // Add to context
            ctx := context.WithValue(r.Context(), SessionKey, sess)
            ctx = context.WithValue(ctx, UserIDKey, sess.UserID)
            r = r.WithContext(ctx)
        }
        
        next.ServeHTTP(w, r)
    })
}

// validateSession extracts and validates session from cookie
func (m *SessionMiddleware) validateSession(r *http.Request) (*session.Session, error) {
    // Extract session cookie
    cookie, err := r.Cookie(m.config.CookieName)
    if err != nil {
        return nil, session.ErrSessionNotFound
    }
    
    sessionID := cookie.Value
    if sessionID == "" {
        return nil, session.ErrInvalidSessionID
    }
    
    // Look up session in database
    sess, err := m.repo.FindByID(r.Context(), sessionID)
    if err != nil {
        return nil, err
    }
    
    // Validate expiration
    if sess.IsExpired(m.config.IdleTimeout) {
        // Delete expired session
        m.repo.Delete(r.Context(), sessionID)
        return nil, session.ErrSessionExpired
    }
    
    // Optional: Validate IP address (disabled by default)
    // if err := sess.ValidateIPAddress(getClientIP(r), false); err != nil {
    //     return nil, err
    // }
    
    return sess, nil
}

// setSessionCookie sets the session cookie
func (m *SessionMiddleware) setSessionCookie(w http.ResponseWriter, sessionID string) {
    http.SetCookie(w, &http.Cookie{
        Name:     m.config.CookieName,
        Value:    sessionID,
        Path:     m.config.CookiePath,
        Domain:   m.config.CookieDomain,
        MaxAge:   int(m.config.MaxAge.Seconds()),
        Secure:   m.config.SecureOnly,
        HttpOnly: true,
        SameSite: sameSiteMode(m.config.SameSite),
    })
}

// clearSessionCookie removes the session cookie
func (m *SessionMiddleware) clearSessionCookie(w http.ResponseWriter) {
    http.SetCookie(w, &http.Cookie{
        Name:     m.config.CookieName,
        Value:    "",
        Path:     m.config.CookiePath,
        Domain:   m.config.CookieDomain,
        MaxAge:   -1,
        Secure:   m.config.SecureOnly,
        HttpOnly: true,
        SameSite: sameSiteMode(m.config.SameSite),
    })
}

// GetSession extracts session from request context
func GetSession(ctx context.Context) (*session.Session, bool) {
    sess, ok := ctx.Value(SessionKey).(*session.Session)
    return sess, ok
}

// Helper functions

func isAPIRequest(r *http.Request) bool {
    return strings.HasPrefix(r.URL.Path, "/api/")
}

func sameSiteMode(mode string) http.SameSite {
    switch mode {
    case "Strict":
        return http.SameSiteStrictMode
    case "Lax":
        return http.SameSiteLaxMode
    case "None":
        return http.SameSiteNoneMode
    default:
        return http.SameSiteLaxMode
    }
}

func getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header (when behind proxy)
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        ips := strings.Split(xff, ",")
        return strings.TrimSpace(ips[0])
    }
    
    // Check X-Real-IP header
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    
    // Fallback to RemoteAddr
    ip, _, _ := net.SplitHostPort(r.RemoteAddr)
    return ip
}
```

#### Login Handler: `internal/infrastructure/http/handlers/auth_handler.go`

```go
package handlers

import (
    "crypto/subtle"
    "net/http"
    "strings"
    
    "github.com/matt-riley/mjrwtf/internal/domain/session"
    "github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// AuthHandler handles authentication operations
type AuthHandler struct {
    sessionRepo session.Repository
    authToken   string
    sessionConfig session.SessionConfig
    sessionMiddleware *middleware.SessionMiddleware
}

func NewAuthHandler(
    sessionRepo session.Repository,
    authToken string,
    sessionConfig session.SessionConfig,
    sessionMiddleware *middleware.SessionMiddleware,
) *AuthHandler {
    return &AuthHandler{
        sessionRepo: sessionRepo,
        authToken:   authToken,
        sessionConfig: sessionConfig,
        sessionMiddleware: sessionMiddleware,
    }
}

// Login handles login form submission
// POST /login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    // Parse form
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }
    
    authToken := strings.TrimSpace(r.FormValue("auth_token"))
    
    // Validate auth token using constant-time comparison
    if subtle.ConstantTimeCompare([]byte(authToken), []byte(h.authToken)) != 1 {
        http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
        return
    }
    
    // Create session
    userID := "authenticated-user" // Future: from user database
    clientIP := middleware.GetClientIP(r)
    userAgent := r.UserAgent()
    
    sess, err := session.NewSession(userID, h.sessionConfig, clientIP, userAgent)
    if err != nil {
        log.Printf("Failed to create session: %v", err)
        http.Error(w, "Failed to create session", http.StatusInternalServerError)
        return
    }
    
    // Store session in database
    if err := h.sessionRepo.Create(r.Context(), sess); err != nil {
        log.Printf("Failed to store session: %v", err)
        http.Error(w, "Failed to store session", http.StatusInternalServerError)
        return
    }
    
    // Set session cookie
    h.sessionMiddleware.setSessionCookie(w, sess.ID)
    
    // Redirect to dashboard
    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Logout handles logout
// POST /logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    // Get session from context
    sess, ok := middleware.GetSession(r.Context())
    if !ok {
        // No session, just redirect
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }
    
    // Delete session from database
    if err := h.sessionRepo.Delete(r.Context(), sess.ID); err != nil {
        log.Printf("Failed to delete session: %v", err)
        // Continue anyway
    }
    
    // Clear session cookie
    h.sessionMiddleware.clearSessionCookie(w)
    
    // Redirect to home
    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

### 7. Updated Route Configuration

#### Modified `server.go` Setup

```go
func (s *Server) setupRoutes() error {
    // ... existing repository initialization ...
    
    // Initialize session repository
    var sessionRepo session.Repository
    if strings.HasPrefix(s.config.DatabaseURL, "postgres://") || 
       strings.HasPrefix(s.config.DatabaseURL, "postgresql://") {
        sessionRepo = repository.NewPostgresSessionRepository(s.db)
    } else {
        sessionRepo = repository.NewSQLiteSessionRepository(s.db)
    }
    
    // Initialize session configuration
    sessionConfig := session.DefaultSessionConfig()
    // Override defaults from environment if needed
    if !s.config.IsProduction {
        sessionConfig.SecureOnly = false // Allow HTTP in development
    }
    
    // Initialize session middleware
    sessionMiddleware := middleware.NewSessionMiddleware(sessionRepo, sessionConfig)
    
    // Initialize auth handler
    authHandler := handlers.NewAuthHandler(
        sessionRepo,
        s.config.AuthToken,
        sessionConfig,
        sessionMiddleware,
    )
    
    // Public routes (no authentication)
    s.router.Get("/", pageHandler.Home)
    s.router.Post("/login", authHandler.Login)
    s.router.Get("/{shortCode}", redirectHandler.Redirect)
    
    // Protected page routes (require session)
    s.router.Group(func(r chi.Router) {
        r.Use(sessionMiddleware.RequireSession)
        r.HandleFunc("/create", pageHandler.CreatePage)
        r.Get("/dashboard", pageHandler.Dashboard)
        r.Post("/logout", authHandler.Logout)
    })
    
    // API routes (bearer token OR session)
    s.router.Route("/api", func(r chi.Router) {
        r.Route("/urls", func(r chi.Router) {
            // Accept either bearer token or session
            r.Use(middleware.AuthOrSession(s.config.AuthToken, sessionMiddleware))
            
            r.Post("/", urlHandler.Create)
            r.Get("/", urlHandler.List)
            r.Delete("/{shortCode}", urlHandler.Delete)
            r.Get("/{shortCode}/analytics", analyticsHandler.GetAnalytics)
        })
    })
    
    // Start session cleanup job
    cleanup := jobs.NewSessionCleanup(sessionRepo, 15*time.Minute)
    cleanup.Start()
    s.cleanupJob = cleanup
    
    return nil
}
```

#### Dual Authentication Middleware: `middleware/auth_or_session.go`

```go
// AuthOrSession accepts either bearer token OR session cookie
// Useful for API endpoints that support both authentication methods
func AuthOrSession(authToken string, sessionMiddleware *SessionMiddleware) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Try bearer token first
            authHeader := r.Header.Get("Authorization")
            if authHeader != "" {
                parts := strings.SplitN(authHeader, " ", 2)
                if len(parts) == 2 && parts[0] == "Bearer" {
                    token := parts[1]
                    if subtle.ConstantTimeCompare([]byte(token), []byte(authToken)) == 1 {
                        // Valid bearer token
                        userID := "authenticated-user"
                        ctx := context.WithValue(r.Context(), UserIDKey, userID)
                        next.ServeHTTP(w, r.WithContext(ctx))
                        return
                    }
                }
            }
            
            // Try session cookie
            sess, err := sessionMiddleware.validateSession(r)
            if err == nil && sess != nil {
                // Valid session
                sess.UpdateActivity()
                sessionMiddleware.repo.UpdateActivity(r.Context(), sess.ID, sess.LastActivity)
                
                ctx := context.WithValue(r.Context(), SessionKey, sess)
                ctx = context.WithValue(ctx, UserIDKey, sess.UserID)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }
            
            // Neither authentication method succeeded
            respondJSONError(w, "Unauthorized: missing or invalid credentials", 
                http.StatusUnauthorized)
        })
    }
}
```

### 8. Configuration Updates

#### Environment Variables: `.env.example`

```bash
# Session Management
SESSION_MAX_AGE=24h              # Maximum session lifetime
SESSION_IDLE_TIMEOUT=30m         # Idle timeout
SESSION_COOKIE_NAME=mjrwtf_session
SESSION_SECURE=true              # Require HTTPS (set false for development)
SESSION_SAME_SITE=Lax            # Lax, Strict, or None
```

#### Config Struct: `internal/infrastructure/config/config.go`

```go
type Config struct {
    // ... existing fields ...
    
    // Session configuration
    SessionMaxAge       time.Duration
    SessionIdleTimeout  time.Duration
    SessionCookieName   string
    SessionSecure       bool
    SessionSameSite     string
    
    // Environment
    IsProduction bool
}

func LoadConfig() (*Config, error) {
    // ... existing code ...
    
    config := &Config{
        // ... existing fields ...
        SessionMaxAge:      parseDuration("SESSION_MAX_AGE", 24*time.Hour),
        SessionIdleTimeout: parseDuration("SESSION_IDLE_TIMEOUT", 30*time.Minute),
        SessionCookieName:  getEnv("SESSION_COOKIE_NAME", "mjrwtf_session"),
        SessionSecure:      getEnvAsBool("SESSION_SECURE", true),
        SessionSameSite:    getEnv("SESSION_SAME_SITE", "Lax"),
        IsProduction:       getEnvAsBool("IS_PRODUCTION", false),
    }
    
    return config, nil
}
```

### 9. Testing Strategy

#### Unit Tests

**Domain Tests:**
- `session_test.go`: Session entity validation
- `csrf_test.go`: CSRF token generation and validation

**Repository Tests:**
- `session_repository_sqlite_test.go`: SQLite session operations
- `session_repository_postgres_test.go`: PostgreSQL session operations

**Middleware Tests:**
- `session_test.go`: Session validation, cookie handling

#### Integration Tests

**Scenarios:**
1. Login flow (create session, set cookie)
2. Authenticated request (validate session, update activity)
3. Session expiration (absolute and idle timeout)
4. Logout (delete session, clear cookie)
5. CSRF protection (token validation)
6. Session fixation prevention
7. Concurrent sessions (multiple devices)

#### Security Tests

**Manual Testing:**
1. Verify httpOnly flag prevents JavaScript access
2. Verify Secure flag requires HTTPS
3. Test CSRF token validation
4. Test session expiration
5. Test logout functionality
6. Test IP validation (if enabled)

**Automated Security Scanning:**
- Use OWASP ZAP or Burp Suite
- Test for session fixation
- Test for session hijacking
- Test CSRF protection

### 10. Migration Path

#### Phase 1: Add Session Infrastructure (No Breaking Changes)

1. Create session domain entities
2. Create database migrations
3. Implement session repository
4. Add session middleware (not yet enforced)
5. Add login/logout handlers
6. Deploy to production

**Result:** New session system available, but old token auth still works

#### Phase 2: Update Frontend (Gradual Migration)

1. Add login page
2. Update frontend to use login flow
3. Remove sessionStorage usage
4. Test with both auth methods working

**Result:** Users can use either auth method

#### Phase 3: Enforce Session Auth (Breaking Change)

1. Update page routes to require session
2. Keep API routes accepting both methods
3. Announce deprecation of token auth
4. Monitor logs for token usage

**Result:** Pages require session, APIs accept both

#### Phase 4: Deprecate Token Auth (Optional)

1. Remove token auth from API endpoints
2. Require all clients to use session auth
3. Remove old auth middleware

**Result:** Single authentication method

### 11. Monitoring and Logging

#### Security Events to Log

```go
// Log session creation
log.Printf("Session created: user=%s ip=%s ua=%s session_id=%s",
    userID, clientIP, userAgent, sessionID)

// Log authentication failures
log.Printf("Authentication failed: ip=%s reason=%s", clientIP, reason)

// Log session expiration
log.Printf("Session expired: session_id=%s user=%s reason=%s",
    sessionID, userID, reason)

// Log suspicious activity
log.Printf("Suspicious activity: session_id=%s user=%s ip=%s event=%s",
    sessionID, userID, clientIP, event)

// Log session cleanup
log.Printf("Session cleanup: deleted=%d", count)
```

#### Metrics to Track

- Active sessions count
- Sessions created per hour
- Authentication failures per hour
- Average session duration
- Sessions expired (absolute vs idle)
- CSRF validation failures

### 12. Future Enhancements

#### User Management
- Replace "authenticated-user" with actual user accounts
- Add user registration and login
- Add password reset flow
- Add email verification

#### Multi-Factor Authentication
- Add TOTP (Google Authenticator)
- Add SMS verification
- Add backup codes

#### Advanced Session Management
- Remember me functionality (longer-lived sessions)
- Device fingerprinting
- Suspicious activity detection
- Geographic anomaly detection

#### OAuth2/OpenID Connect
- Social login (Google, GitHub, Discord)
- API access tokens with scopes
- Refresh tokens

## Summary

This design provides a comprehensive, secure session management system that:

1. **Eliminates XSS Risk**: httpOnly cookies prevent JavaScript access
2. **Prevents CSRF**: SameSite=Lax + CSRF tokens
3. **Prevents Session Fixation**: Fresh session ID on login
4. **Prevents Session Hijacking**: HTTPS, Secure flag, security headers
5. **Limits Attack Window**: Configurable timeouts
6. **Enables Logout**: User-controlled session lifecycle
7. **Maintains Architecture**: Follows hexagonal patterns
8. **Supports Both Databases**: SQLite and PostgreSQL
9. **Production Ready**: Comprehensive security controls
10. **Backwards Compatible**: Gradual migration path

The implementation prioritizes security while maintaining usability and follows OWASP best practices throughout.
