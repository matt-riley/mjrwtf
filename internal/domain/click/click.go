package click

import (
	"net/url"
	"strings"
	"time"
)

// Click represents a click event on a shortened URL for analytics
type Click struct {
	ID             int64
	URLID          int64
	ClickedAt      time.Time
	Referrer       string
	ReferrerDomain string
	Country        string
	UserAgent      string
}

// NewClick creates a new Click with validation
func NewClick(urlID int64, referrer, country, userAgent string) (*Click, error) {
	// Normalize country code by trimming spaces
	country = strings.TrimSpace(country)

	// Extract domain from referrer URL
	referrerDomain := extractDomain(referrer)

	c := &Click{
		URLID:          urlID,
		ClickedAt:      time.Now(),
		Referrer:       referrer,
		ReferrerDomain: referrerDomain,
		Country:        country,
		UserAgent:      userAgent,
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// extractDomain extracts the hostname from a referrer URL.
// Returns empty string if referrer is empty or malformed.
func extractDomain(referrer string) string {
	if referrer == "" {
		return ""
	}

	// Parse the URL
	parsedURL, err := url.Parse(referrer)
	if err != nil {
		// Malformed URL, return empty string
		return ""
	}

	// Return the hostname (excluding port if present)
	// This handles http, https, and other schemes
	return parsedURL.Hostname()
}

// Validate validates the Click entity
func (c *Click) Validate() error {
	if c.URLID <= 0 {
		return ErrInvalidURLID
	}

	// Country code must be empty or exactly 2 characters (ISO 3166-1 alpha-2)
	if c.Country != "" && len(c.Country) != 2 {
		return ErrInvalidCountryCode
	}

	return nil
}
