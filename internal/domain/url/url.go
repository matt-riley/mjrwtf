package url

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// URL represents a shortened URL in the domain
type URL struct {
	ID          int64
	ShortCode   string
	OriginalURL string
	CreatedAt   time.Time
	CreatedBy   string
}

var (
	// shortCodeRegex validates short codes: alphanumeric characters, underscores, hyphens, 3-20 characters
	shortCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,20}$`)
)

// NewURL creates a new URL with validation
func NewURL(shortCode, originalURL, createdBy string) (*URL, error) {
	u := &URL{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	if err := u.Validate(); err != nil {
		return nil, err
	}

	return u, nil
}

// Validate validates the URL entity
func (u *URL) Validate() error {
	if err := ValidateShortCode(u.ShortCode); err != nil {
		return err
	}

	if err := ValidateOriginalURL(u.OriginalURL); err != nil {
		return err
	}

	if strings.TrimSpace(u.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}

	return nil
}

// ValidateShortCode validates a short code format
func ValidateShortCode(shortCode string) error {
	if shortCode == "" {
		return ErrEmptyShortCode
	}

	if !shortCodeRegex.MatchString(shortCode) {
		return ErrInvalidShortCode
	}

	return nil
}

// ValidateOriginalURL validates an original URL
func ValidateOriginalURL(originalURL string) error {
	if originalURL == "" {
		return ErrEmptyOriginalURL
	}

	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidOriginalURL, err)
	}

	// URL must have a scheme (http/https)
	if parsedURL.Scheme == "" {
		return ErrMissingURLScheme
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrInvalidURLScheme
	}

	// URL must have a host
	if parsedURL.Host == "" {
		return ErrMissingURLHost
	}

	return nil
}
