package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// CreateURLRequest represents the input for creating a shortened URL
type CreateURLRequest struct {
	OriginalURL string
	CreatedBy   string
}

// CreateURLResponse represents the output after creating a shortened URL
type CreateURLResponse struct {
	ShortCode   string
	ShortURL    string
	OriginalURL string
}

// CreateURLUseCase handles the creation of shortened URLs
type CreateURLUseCase struct {
	generator *url.Generator
	baseURL   string
}

// NewCreateURLUseCase creates a new CreateURLUseCase
func NewCreateURLUseCase(generator *url.Generator, baseURL string) *CreateURLUseCase {
	return &CreateURLUseCase{
		generator: generator,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
	}
}

// Execute creates a shortened URL
func (uc *CreateURLUseCase) Execute(ctx context.Context, req CreateURLRequest) (*CreateURLResponse, error) {
	// Generate and store shortened URL
	shortenedURL, err := uc.generator.ShortenURL(ctx, req.OriginalURL, req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create shortened URL: %w", err)
	}

	// Build response
	shortURL := fmt.Sprintf("%s/%s", uc.baseURL, shortenedURL.ShortCode)

	return &CreateURLResponse{
		ShortCode:   shortenedURL.ShortCode,
		ShortURL:    shortURL,
		OriginalURL: shortenedURL.OriginalURL,
	}, nil
}
