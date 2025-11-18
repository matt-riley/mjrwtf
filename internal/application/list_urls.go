package application

import (
	"context"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// ListURLsRequest represents the input for listing URLs
type ListURLsRequest struct {
	CreatedBy string
	Limit     int
	Offset    int
}

// URLResponse represents a single URL in the response
type URLResponse struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

// ListURLsResponse represents the output after listing URLs
type ListURLsResponse struct {
	URLs   []URLResponse `json:"urls"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// ListURLsUseCase handles listing of shortened URLs
type ListURLsUseCase struct {
	urlRepo url.Repository
}

// NewListURLsUseCase creates a new ListURLsUseCase
func NewListURLsUseCase(urlRepo url.Repository) *ListURLsUseCase {
	return &ListURLsUseCase{
		urlRepo: urlRepo,
	}
}

// Execute lists URLs for a specific user with pagination
func (uc *ListURLsUseCase) Execute(ctx context.Context, req ListURLsRequest) (*ListURLsResponse, error) {
	// Validate requested by
	if req.CreatedBy == "" {
		return nil, url.ErrInvalidCreatedBy
	}

	// Set default values for pagination
	limit := req.Limit
	if limit <= 0 {
		limit = 20 // Default limit
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// Retrieve URLs from repository
	urls, err := uc.urlRepo.List(ctx, req.CreatedBy, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert domain URLs to response format
	urlResponses := make([]URLResponse, len(urls))
	for i, u := range urls {
		urlResponses[i] = URLResponse{
			ShortCode:   u.ShortCode,
			OriginalURL: u.OriginalURL,
			CreatedAt:   u.CreatedAt,
			CreatedBy:   u.CreatedBy,
		}
	}

	return &ListURLsResponse{
		URLs:   urlResponses,
		Total:  len(urlResponses),
		Limit:  limit,
		Offset: offset,
	}, nil
}
