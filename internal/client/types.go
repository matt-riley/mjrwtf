package client

import "time"

type CreateURLRequest struct {
	OriginalURL string `json:"original_url"`
}

type CreateURLResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type URLResponse struct {
	ID          int64     `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	ClickCount  int64     `json:"click_count"`
}

type ListURLsResponse struct {
	URLs   []URLResponse `json:"urls"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type GetAnalyticsResponse struct {
	ShortCode   string           `json:"short_code"`
	OriginalURL string           `json:"original_url"`
	TotalClicks int64            `json:"total_clicks"`
	ByCountry   map[string]int64 `json:"by_country"`
	ByReferrer  map[string]int64 `json:"by_referrer"`
	ByDate      map[string]int64 `json:"by_date,omitempty"`
	StartTime   *time.Time       `json:"start_time,omitempty"`
	EndTime     *time.Time       `json:"end_time,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
