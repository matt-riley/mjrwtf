package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Session represents an authenticated user session in the domain
type Session struct {
	ID             string
	UserID         string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	LastActivityAt time.Time
	IPAddress      string
	UserAgent      string
}

var (
	// sessionIDRegex validates session IDs: exactly 64 hexadecimal characters
	sessionIDRegex = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

const (
	// SessionIDLength is the length of a session ID in hexadecimal characters
	SessionIDLength = 64
	// SessionIDBytes is the number of random bytes needed for a session ID
	SessionIDBytes = 32
	// UserAgentMaxLength is the maximum allowed length for a user agent string
	UserAgentMaxLength = 500
	// DefaultSessionDuration is the default session duration (24 hours)
	DefaultSessionDuration = 24 * time.Hour
)

// NewSession creates a new session with a cryptographically secure random ID
func NewSession(userID, ipAddress, userAgent string) (*Session, error) {
	// Generate cryptographically secure random session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	s := &Session{
		ID:             sessionID,
		UserID:         userID,
		CreatedAt:      now,
		ExpiresAt:      now.Add(DefaultSessionDuration),
		LastActivityAt: now,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	return s, nil
}

// generateSessionID generates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, SessionIDBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsIdle checks if the session has been idle longer than the specified timeout
func (s *Session) IsIdle(idleTimeout time.Duration) bool {
	return time.Since(s.LastActivityAt) > idleTimeout
}

// UpdateActivity updates the session's last activity timestamp to the current time
func (s *Session) UpdateActivity() {
	s.LastActivityAt = time.Now()
}

// Validate validates the Session entity
func (s *Session) Validate() error {
	if err := ValidateSessionID(s.ID); err != nil {
		return err
	}

	if strings.TrimSpace(s.UserID) == "" {
		return ErrEmptyUserID
	}

	if !s.ExpiresAt.After(s.CreatedAt) {
		return ErrInvalidTimestamps
	}

	if s.LastActivityAt.Before(s.CreatedAt) {
		return ErrInvalidTimestamps
	}

	if len(s.UserAgent) > UserAgentMaxLength {
		return ErrUserAgentTooLong
	}

	return nil
}

// ValidateSessionID validates a session ID format
func ValidateSessionID(sessionID string) error {
	if sessionID == "" {
		return ErrInvalidSessionID
	}

	if !sessionIDRegex.MatchString(sessionID) {
		return ErrInvalidSessionID
	}

	return nil
}
