package application

import (
	"context"
	"fmt"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// DeleteURLRequest represents the input for deleting a shortened URL
type DeleteURLRequest struct {
	ShortCode   string
	RequestedBy string
}

// DeleteURLResponse represents the output after deleting a shortened URL
type DeleteURLResponse struct {
	Success bool
}

// DeleteURLUseCase handles the deletion of shortened URLs with authorization
type DeleteURLUseCase struct {
	urlRepo url.Repository
}

// NewDeleteURLUseCase creates a new DeleteURLUseCase
func NewDeleteURLUseCase(urlRepo url.Repository) *DeleteURLUseCase {
	return &DeleteURLUseCase{
		urlRepo: urlRepo,
	}
}

// Execute deletes a shortened URL after verifying ownership
func (uc *DeleteURLUseCase) Execute(ctx context.Context, req DeleteURLRequest) (*DeleteURLResponse, error) {
	// Validate short code
	if err := url.ValidateShortCode(req.ShortCode); err != nil {
		return nil, fmt.Errorf("invalid short code: %w", err)
	}

	// Validate requested by
	if req.RequestedBy == "" {
		return nil, url.ErrInvalidCreatedBy
	}

	// Find the URL to verify it exists and check ownership
	foundURL, err := uc.urlRepo.FindByShortCode(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	// Verify ownership - only the creator can delete the URL
	if foundURL.CreatedBy != req.RequestedBy {
		return nil, url.ErrUnauthorizedDeletion
	}

	// Delete the URL from the repository
	if err := uc.urlRepo.Delete(ctx, req.ShortCode); err != nil {
		return nil, fmt.Errorf("failed to delete URL: %w", err)
	}

	return &DeleteURLResponse{
		Success: true,
	}, nil
}
