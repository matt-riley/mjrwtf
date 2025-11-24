package session

import "errors"

// Domain errors for Session operations
var (
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when a session has expired
	ErrSessionExpired = errors.New("session has expired")

	// ErrSessionIdle is returned when a session has been idle too long
	ErrSessionIdle = errors.New("session has been idle too long")

	// ErrInvalidSessionID is returned when a session ID format is invalid
	ErrInvalidSessionID = errors.New("session ID must be 64 hexadecimal characters")

	// ErrEmptyUserID is returned when a user ID is empty
	ErrEmptyUserID = errors.New("user ID cannot be empty")

	// ErrInvalidTimestamps is returned when timestamps are invalid
	ErrInvalidTimestamps = errors.New("expires_at must be after created_at, and last_activity_at must be >= created_at")

	// ErrUserAgentTooLong is returned when a user agent exceeds maximum length
	ErrUserAgentTooLong = errors.New("user agent cannot exceed 500 characters")
)
