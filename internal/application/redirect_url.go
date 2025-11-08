package application

import (
	"context"
	"log"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// RedirectRequest contains the data needed to redirect and track a short URL
type RedirectRequest struct {
	ShortCode string
	Referrer  string
	UserAgent string
	IPAddress string
	Country   string
}

// RedirectResponse contains the result of a redirect lookup
type RedirectResponse struct {
	OriginalURL string
}

// RedirectURLUseCase handles redirecting short URLs and tracking analytics
type RedirectURLUseCase struct {
	urlRepo   url.Repository
	clickRepo click.Repository
}

// NewRedirectURLUseCase creates a new RedirectURLUseCase
func NewRedirectURLUseCase(urlRepo url.Repository, clickRepo click.Repository) *RedirectURLUseCase {
	return &RedirectURLUseCase{
		urlRepo:   urlRepo,
		clickRepo: clickRepo,
	}
}

// Execute performs the redirect lookup and records analytics asynchronously
func (uc *RedirectURLUseCase) Execute(ctx context.Context, req RedirectRequest) (*RedirectResponse, error) {
	// Look up URL by short code
	foundURL, err := uc.urlRepo.FindByShortCode(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	// Record click asynchronously (non-blocking)
	go func() {
		// Use a background context for the async operation
		// so it's not cancelled when the request context is cancelled
		bgCtx := context.Background()
		
		newClick, err := click.NewClick(foundURL.ID, req.Referrer, req.Country, req.UserAgent)
		if err != nil {
			log.Printf("Failed to create click entity for URL %s: %v", req.ShortCode, err)
			return
		}

		if err := uc.clickRepo.Record(bgCtx, newClick); err != nil {
			log.Printf("Failed to record click for URL %s: %v", req.ShortCode, err)
		}
	}()

	return &RedirectResponse{
		OriginalURL: foundURL.OriginalURL,
	}, nil
}
