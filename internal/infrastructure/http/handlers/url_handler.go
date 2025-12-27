package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// CreateURLUseCase defines the interface for creating URLs
type CreateURLUseCase interface {
	Execute(ctx context.Context, req application.CreateURLRequest) (*application.CreateURLResponse, error)
}

// ListURLsUseCase defines the interface for listing URLs
type ListURLsUseCase interface {
	Execute(ctx context.Context, req application.ListURLsRequest) (*application.ListURLsResponse, error)
}

// DeleteURLUseCase defines the interface for deleting URLs
type DeleteURLUseCase interface {
	Execute(ctx context.Context, req application.DeleteURLRequest) (*application.DeleteURLResponse, error)
}

// URLHandler handles HTTP requests for URL operations
type URLHandler struct {
	createUseCase CreateURLUseCase
	listUseCase   ListURLsUseCase
	deleteUseCase DeleteURLUseCase
}

// NewURLHandler creates a new URLHandler
func NewURLHandler(
	createUseCase CreateURLUseCase,
	listUseCase ListURLsUseCase,
	deleteUseCase DeleteURLUseCase,
) *URLHandler {
	return &URLHandler{
		createUseCase: createUseCase,
		listUseCase:   listUseCase,
		deleteUseCase: deleteUseCase,
	}
}

// CreateURLRequest represents the JSON request body for creating a URL
type CreateURLRequest struct {
	OriginalURL string `json:"original_url"`
}

// CreateURLResponse represents the JSON response for creating a URL
type CreateURLResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// Create handles POST /api/urls - Create shortened URL
func (h *URLHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body (strict JSON + size limits)
	var req CreateURLRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		respondJSONDecodeError(w, err)
		return
	}

	// Validate request
	if req.OriginalURL == "" {
		respondError(w, "original_url is required", http.StatusBadRequest)
		return
	}

	// Execute use case
	resp, err := h.createUseCase.Execute(r.Context(), application.CreateURLRequest{
		OriginalURL: req.OriginalURL,
		CreatedBy:   userID,
	})

	if err != nil {
		handleDomainError(w, err)
		return
	}

	// Respond with success
	respondJSON(w, CreateURLResponse{
		ShortCode:   resp.ShortCode,
		ShortURL:    resp.ShortURL,
		OriginalURL: resp.OriginalURL,
	}, http.StatusCreated)
}

// List handles GET /api/urls - List user's URLs
func (h *URLHandler) List(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters for pagination
	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)

	// Validate pagination parameters
	if limit < 0 || offset < 0 {
		respondError(w, "limit and offset must be non-negative", http.StatusBadRequest)
		return
	}
	// Execute use case
	resp, err := h.listUseCase.Execute(r.Context(), application.ListURLsRequest{
		CreatedBy: userID,
		Limit:     limit,
		Offset:    offset,
	})

	if err != nil {
		handleDomainError(w, err)
		return
	}

	// Respond with success
	respondJSON(w, resp, http.StatusOK)
}

// Delete handles DELETE /api/urls/:shortCode - Delete URL
func (h *URLHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	// Execute use case
	_, err := h.deleteUseCase.Execute(r.Context(), application.DeleteURLRequest{
		ShortCode:   shortCode,
		RequestedBy: userID,
	})

	if err != nil {
		handleDomainError(w, err)
		return
	}

	// Respond with no content on success
	w.WriteHeader(http.StatusNoContent)
}
