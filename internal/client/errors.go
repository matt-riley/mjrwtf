package client

import (
	"fmt"
	"time"
)

// APIError represents a non-success API response.
//
// For HTTP 429 responses, RetryAfter will be set when the server provides a valid Retry-After header.
type APIError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e == nil {
		return "api error"
	}
	if e.StatusCode == 0 {
		return e.Message
	}
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
}
