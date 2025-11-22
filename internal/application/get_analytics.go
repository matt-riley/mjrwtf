package application

import (
	"context"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// GetAnalyticsRequest represents the input for getting analytics for a URL
type GetAnalyticsRequest struct {
	ShortCode string
	StartTime *time.Time // Optional: filter clicks from this time
	EndTime   *time.Time // Optional: filter clicks until this time
}

// GetAnalyticsResponse represents the analytics data for a URL
type GetAnalyticsResponse struct {
	ShortCode   string              `json:"short_code"`
	OriginalURL string              `json:"original_url"`
	TotalClicks int64               `json:"total_clicks"`
	ByCountry   map[string]int64    `json:"by_country"`
	ByReferrer  map[string]int64    `json:"by_referrer"`
	ByDate      map[string]int64    `json:"by_date,omitempty"` // Only for all-time stats
	StartTime   *time.Time          `json:"start_time,omitempty"`
	EndTime     *time.Time          `json:"end_time,omitempty"`
}

// GetAnalyticsUseCase handles retrieving analytics for shortened URLs
type GetAnalyticsUseCase struct {
	urlRepo   url.Repository
	clickRepo click.Repository
}

// NewGetAnalyticsUseCase creates a new GetAnalyticsUseCase
func NewGetAnalyticsUseCase(urlRepo url.Repository, clickRepo click.Repository) *GetAnalyticsUseCase {
	return &GetAnalyticsUseCase{
		urlRepo:   urlRepo,
		clickRepo: clickRepo,
	}
}

// Execute retrieves analytics data for a specific URL
func (uc *GetAnalyticsUseCase) Execute(ctx context.Context, req GetAnalyticsRequest) (*GetAnalyticsResponse, error) {
	// Validate short code
	if req.ShortCode == "" {
		return nil, url.ErrEmptyShortCode
	}

	// Find URL by short code
	foundURL, err := uc.urlRepo.FindByShortCode(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	// Check if time range is specified
	if req.StartTime != nil && req.EndTime != nil {
		// Get time range statistics
		stats, err := uc.clickRepo.GetStatsByURLAndTimeRange(ctx, foundURL.ID, *req.StartTime, *req.EndTime)
		if err != nil {
			return nil, err
		}

		return &GetAnalyticsResponse{
			ShortCode:   foundURL.ShortCode,
			OriginalURL: foundURL.OriginalURL,
			TotalClicks: stats.TotalCount,
			ByCountry:   stats.ByCountry,
			ByReferrer:  stats.ByReferrer,
			StartTime:   req.StartTime,
			EndTime:     req.EndTime,
		}, nil
	}

	// Get all-time statistics
	stats, err := uc.clickRepo.GetStatsByURL(ctx, foundURL.ID)
	if err != nil {
		return nil, err
	}

	return &GetAnalyticsResponse{
		ShortCode:   foundURL.ShortCode,
		OriginalURL: foundURL.OriginalURL,
		TotalClicks: stats.TotalCount,
		ByCountry:   stats.ByCountry,
		ByReferrer:  stats.ByReferrer,
		ByDate:      stats.ByDate,
	}, nil
}
