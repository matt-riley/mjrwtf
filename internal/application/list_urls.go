package application

import (
	"context"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
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
	ID          int64     `json:"id"`           // ID is the unique identifier of the URL
	ShortCode   string    `json:"short_code"`   // ShortCode is the shortened URL identifier
	OriginalURL string    `json:"original_url"` // OriginalURL is the original long URL
	CreatedAt   time.Time `json:"created_at"`   // CreatedAt is when the URL was created
	CreatedBy   string    `json:"created_by"`   // CreatedBy is the user who created the URL
	ClickCount  int64     `json:"click_count"`  // ClickCount is the total number of clicks on this URL
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
	urlRepo   url.Repository
	clickRepo click.Repository
}

// NewListURLsUseCase creates a new ListURLsUseCase
func NewListURLsUseCase(urlRepo url.Repository, clickRepo click.Repository) *ListURLsUseCase {
	return &ListURLsUseCase{
		urlRepo:   urlRepo,
		clickRepo: clickRepo,
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

	// Convert domain URLs to response format and fetch click counts
	urlResponses := make([]URLResponse, len(urls))
	for i, u := range urls {
		// Fetch click count for this URL
		clickCount, err := uc.clickRepo.GetTotalClickCount(ctx, u.ID)
		if err != nil {
			// Default to 0 if we can't get the count
			clickCount = 0
		}

		urlResponses[i] = URLResponse{
			ID:          u.ID,
			ShortCode:   u.ShortCode,
			OriginalURL: u.OriginalURL,
			CreatedAt:   u.CreatedAt,
			CreatedBy:   u.CreatedBy,
			ClickCount:  clickCount,
		}
	}

	// FIXME: Pagination Total field currently returns page count, not total count
	// The Total field should represent the total count of URLs across all pages,
	// but the Repository interface doesn't have a Count() method yet.
	// For now, we return the count of URLs in the current page as a temporary workaround.
	// This is a known limitation that breaks proper pagination semantics.
	// See: Repository interface needs Count(ctx, createdBy) method
	totalCount := len(urls)

	return &ListURLsResponse{
		URLs:   urlResponses,
		Total:  totalCount,
		Limit:  limit,
		Offset: offset,
	}, nil
}
