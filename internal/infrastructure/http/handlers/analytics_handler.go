package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
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

// GetAnalytics handles GET /api/urls/{shortCode}/analytics - Get analytics for a URL
func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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

	// Validate that start_time is strictly before end_time (equality not allowed)
	if startTime != nil && endTime != nil && !startTime.Before(*endTime) {
		respondError(w, "start_time must be strictly before end_time (equality not allowed)", http.StatusBadRequest)
		return
	}

	// Execute use case
	resp, err := h.getAnalyticsUseCase.Execute(r.Context(), application.GetAnalyticsRequest{
		ShortCode:   shortCode,
		RequestedBy: userID,
		StartTime:   startTime,
		EndTime:     endTime,
	})

	if err != nil {
		handleDomainError(w, err)
		return
	}

	// Respond with success
	respondJSON(w, resp, http.StatusOK)
}
