package url

import "errors"

// Domain errors for URL operations
var (
	// ErrURLNotFound is returned when a URL is not found by its short code
	ErrURLNotFound = errors.New("url not found")

	// ErrDuplicateShortCode is returned when attempting to create a URL with an existing short code
	ErrDuplicateShortCode = errors.New("short code already exists")

	// ErrEmptyShortCode is returned when a short code is empty
	ErrEmptyShortCode = errors.New("short code cannot be empty")

	// ErrInvalidShortCode is returned when a short code format is invalid
	ErrInvalidShortCode = errors.New("short code must be 3-20 characters long and contain only alphanumeric characters, underscores, or hyphens")

	// ErrEmptyOriginalURL is returned when an original URL is empty
	ErrEmptyOriginalURL = errors.New("original URL cannot be empty")

	// ErrInvalidOriginalURL is returned when an original URL format is invalid
	ErrInvalidOriginalURL = errors.New("invalid original URL format")

	// ErrMissingURLScheme is returned when a URL doesn't have a scheme
	ErrMissingURLScheme = errors.New("URL must have a scheme (http or https)")

	// ErrInvalidURLScheme is returned when a URL has an unsupported scheme
	ErrInvalidURLScheme = errors.New("URL scheme must be http or https")

	// ErrMissingURLHost is returned when a URL doesn't have a host
	ErrMissingURLHost = errors.New("URL must have a host")

	// ErrInvalidCreatedBy is returned when created_by is empty
	ErrInvalidCreatedBy = errors.New("created_by cannot be empty")
)
