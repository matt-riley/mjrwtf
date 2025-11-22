package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// GetAnalyticsUseCase defines the interface for getting analytics
type GetAnalyticsUseCase interface {
	Execute(ctx context.Context, req application.GetAnalyticsRequest) (*application.GetAnalyticsResponse, error)
}

// AnalyticsHandler handles HTTP requests for analytics operations
type AnalyticsHandler struct {
	getAnalyticsUseCase GetAnalyticsUseCase
}

// NewAnalyticsHandler creates a new AnalyticsHandler
func NewAnalyticsHandler(getAnalyticsUseCase GetAnalyticsUseCase) *AnalyticsHandler {
	return &AnalyticsHandler{
		getAnalyticsUseCase: getAnalyticsUseCase,
	}
}

// GetAnalytics handles GET /api/urls/:shortCode/analytics - Get analytics for a URL
func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	// Extract short code from URL path
	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		respondError(w, "short_code is required", http.StatusBadRequest)
		return
	}

	// Parse optional time range parameters
	var startTime, endTime *time.Time
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			respondError(w, "invalid start_time format, use RFC3339 (e.g., 2025-11-20T00:00:00Z)", http.StatusBadRequest)
			return
		}
		startTime = &t
	}

	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			respondError(w, "invalid end_time format, use RFC3339 (e.g., 2025-11-22T23:59:59Z)", http.StatusBadRequest)
			return
		}
		endTime = &t
	}

	// Both start_time and end_time must be provided together
	if (startTime != nil && endTime == nil) || (startTime == nil && endTime != nil) {
		respondError(w, "both start_time and end_time must be provided for time range queries", http.StatusBadRequest)
		return
	}

	// Execute use case
	resp, err := h.getAnalyticsUseCase.Execute(r.Context(), application.GetAnalyticsRequest{
		ShortCode: shortCode,
		StartTime: startTime,
		EndTime:   endTime,
	})

	if err != nil {
		handleAnalyticsError(w, err)
		return
	}

	// Respond with success
	respondJSON(w, resp, http.StatusOK)
}

// handleAnalyticsError maps domain errors to HTTP status codes
func handleAnalyticsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, url.ErrURLNotFound):
		respondError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, url.ErrEmptyShortCode):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrInvalidShortCode):
		respondError(w, err.Error(), http.StatusBadRequest)
	default:
		respondError(w, "internal server error", http.StatusInternalServerError)
	}
}
