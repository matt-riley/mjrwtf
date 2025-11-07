package click

import "errors"

// Domain errors for Click operations
var (
	// ErrInvalidURLID is returned when a URL ID is invalid (zero or negative)
	ErrInvalidURLID = errors.New("URL ID must be positive")

	// ErrInvalidCountryCode is returned when a country code is not 2 characters
	ErrInvalidCountryCode = errors.New("country code must be empty or exactly 2 characters (ISO 3166-1 alpha-2)")

	// ErrClickNotFound is returned when a click is not found
	ErrClickNotFound = errors.New("click not found")
)
