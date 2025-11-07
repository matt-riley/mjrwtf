package click

import (
	"strings"
	"time"
)

// Click represents a click event on a shortened URL for analytics
type Click struct {
	ID        int64
	URLID     int64
	ClickedAt time.Time
	Referrer  string
	Country   string
	UserAgent string
}

// NewClick creates a new Click with validation
func NewClick(urlID int64, referrer, country, userAgent string) (*Click, error) {
	c := &Click{
		URLID:     urlID,
		ClickedAt: time.Now(),
		Referrer:  referrer,
		Country:   country,
		UserAgent: userAgent,
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// Validate validates the Click entity
func (c *Click) Validate() error {
	if c.URLID <= 0 {
		return ErrInvalidURLID
	}

	// Country code must be empty or exactly 2 characters (ISO 3166-1 alpha-2)
	if c.Country != "" && len(strings.TrimSpace(c.Country)) != 2 {
		return ErrInvalidCountryCode
	}

	return nil
}
