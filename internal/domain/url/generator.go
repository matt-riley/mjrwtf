package url

import (
	"crypto/rand"
	"errors"
	"math/big"
)

// Base62 character set for short code generation
const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (
	// ErrMaxRetriesExceeded is returned when collision resolution fails after maximum retries
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded for short code generation")

	// ErrInvalidCodeLength is returned when code length is invalid
	ErrInvalidCodeLength = errors.New("code length must be between 3 and 20 characters")
)

// Generator generates short codes for URLs
type Generator struct {
	codeLength int
	maxRetries int
	repository Repository
}

// GeneratorConfig holds configuration for the Generator
type GeneratorConfig struct {
	// CodeLength is the length of generated short codes (default: 6)
	CodeLength int
	// MaxRetries is the maximum number of retry attempts for collision resolution (default: 3)
	MaxRetries int
}

// DefaultGeneratorConfig returns the default configuration
func DefaultGeneratorConfig() GeneratorConfig {
	return GeneratorConfig{
		CodeLength: 6,
		MaxRetries: 3,
	}
}

// NewGenerator creates a new Generator with the given repository and config
func NewGenerator(repo Repository, config GeneratorConfig) (*Generator, error) {
	if config.CodeLength < 3 || config.CodeLength > 20 {
		return nil, ErrInvalidCodeLength
	}

	if config.MaxRetries < 1 {
		config.MaxRetries = 1
	}

	return &Generator{
		codeLength: config.CodeLength,
		maxRetries: config.MaxRetries,
		repository: repo,
	}, nil
}

// GenerateShortCode generates a random base62 short code
func (g *Generator) GenerateShortCode() (string, error) {
	code := make([]byte, g.codeLength)
	charsetLen := big.NewInt(int64(len(base62Chars)))

	for i := 0; i < g.codeLength; i++ {
		// Use crypto/rand for cryptographically secure random number generation
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		code[i] = base62Chars[randomIndex.Int64()]
	}

	return string(code), nil
}

// GenerateUniqueShortCode generates a unique short code with collision detection
func (g *Generator) GenerateUniqueShortCode() (string, error) {
	for attempt := 0; attempt < g.maxRetries; attempt++ {
		code, err := g.GenerateShortCode()
		if err != nil {
			return "", err
		}

		// Check for collision by attempting to find existing URL with this code
		_, err = g.repository.FindByShortCode(code)
		if errors.Is(err, ErrURLNotFound) {
			// No collision, code is unique
			return code, nil
		}
		if err != nil {
			// Some other error occurred
			return "", err
		}

		// Collision detected, retry
	}

	return "", ErrMaxRetriesExceeded
}

// ShortenURL creates a shortened URL with a unique short code
func (g *Generator) ShortenURL(originalURL, createdBy string) (*URL, error) {
	// Validate URL before generating short code
	if err := ValidateOriginalURL(originalURL); err != nil {
		return nil, err
	}

	// Generate unique short code
	shortCode, err := g.GenerateUniqueShortCode()
	if err != nil {
		return nil, err
	}

	// Create URL entity
	url, err := NewURL(shortCode, originalURL, createdBy)
	if err != nil {
		return nil, err
	}

	// Persist to repository
	if err := g.repository.Create(url); err != nil {
		return nil, err
	}

	return url, nil
}
