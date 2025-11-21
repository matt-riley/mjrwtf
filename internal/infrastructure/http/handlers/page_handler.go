package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// PageHandler handles HTML page rendering
type PageHandler struct {
	createUseCase CreateURLUseCase
	listUseCase   ListURLsUseCase
	urlRepo       url.Repository
	clickRepo     click.Repository
	authToken     string
}

// NewPageHandler creates a new PageHandler
func NewPageHandler(
	createUseCase CreateURLUseCase,
	listUseCase ListURLsUseCase,
	urlRepo url.Repository,
	clickRepo click.Repository,
	authToken string,
) *PageHandler {
	return &PageHandler{
		createUseCase: createUseCase,
		listUseCase:   listUseCase,
		urlRepo:       urlRepo,
		clickRepo:     clickRepo,
		authToken:     authToken,
	}
}

// Home renders the home page
func (h *PageHandler) Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := pages.Home().Render(r.Context(), w); err != nil {
		// Note: Status already written, just log the error
		// Template rendering errors are rare and indicate a serious issue
	}
}

// NotFound renders the 404 error page
func (h *PageHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := pages.NotFound().Render(r.Context(), w); err != nil {
		// Fallback to plain text if template fails
		w.Write([]byte("Error rendering page"))
	}
}

// InternalError renders the 500 error page
func (h *PageHandler) InternalError(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	if err := pages.InternalError(message).Render(r.Context(), w); err != nil {
		// Fallback to plain text if template fails
		w.Write([]byte(message))
	}
}

// CreatePage renders the URL creation form page
func (h *PageHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	// Handle GET request - show empty form
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		if err := pages.Create().Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	// Handle POST request - process form submission
	if r.Method == http.MethodPost {
		h.handleCreateURLForm(w, r)
		return
	}
	
	// Method not allowed
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// handleCreateURLForm processes the URL creation form submission
func (h *PageHandler) handleCreateURLForm(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := pages.CreateWithError("Invalid form data").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	// Extract form values
	originalURL := strings.TrimSpace(r.FormValue("original_url"))
	authToken := strings.TrimSpace(r.FormValue("auth_token"))
	
	// Server-side validation
	if originalURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := pages.CreateWithError("URL is required").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	if authToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := pages.CreateWithError("Authentication token is required").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	// Validate auth token using constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(authToken), []byte(h.authToken)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		if err := pages.CreateWithError("Invalid authentication token").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	// Create context with user ID (use typed context key for type safety)
	userID := "authenticated-user"
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	
	// Call the create URL use case
	resp, err := h.createUseCase.Execute(ctx, application.CreateURLRequest{
		OriginalURL: originalURL,
		CreatedBy:   userID,
	})
	
	if err != nil {
		// Map domain errors to user-friendly messages
		h.renderErrorPage(w, r, err)
		return
	}
	
	// Render success page with result
	w.WriteHeader(http.StatusOK)
	if err := pages.CreateWithResult(resp.ShortCode, resp.ShortURL, resp.OriginalURL).Render(r.Context(), w); err != nil {
		w.Write([]byte("Error rendering page"))
	}
}

// renderErrorPage maps domain errors to appropriate HTTP status codes and user-friendly messages
func (h *PageHandler) renderErrorPage(w http.ResponseWriter, r *http.Request, err error) {
	var statusCode int
	var errorMsg string
	
	// Map domain errors to HTTP status codes and messages
	switch {
	case errors.Is(err, url.ErrInvalidOriginalURL):
		statusCode = http.StatusBadRequest
		errorMsg = "Invalid URL format"
	case errors.Is(err, url.ErrEmptyOriginalURL):
		statusCode = http.StatusBadRequest
		errorMsg = "URL is required"
	case errors.Is(err, url.ErrMissingURLScheme):
		statusCode = http.StatusBadRequest
		errorMsg = "URL must include http:// or https://"
	case errors.Is(err, url.ErrInvalidURLScheme):
		statusCode = http.StatusBadRequest
		errorMsg = "URL must use HTTP or HTTPS protocol"
	case errors.Is(err, url.ErrMissingURLHost):
		statusCode = http.StatusBadRequest
		errorMsg = "URL must include a valid host"
	case errors.Is(err, url.ErrInvalidShortCode):
		statusCode = http.StatusBadRequest
		errorMsg = "Invalid short code format"
	case errors.Is(err, url.ErrDuplicateShortCode):
		statusCode = http.StatusConflict
		errorMsg = "This short code is already in use, please try again"
	default:
		statusCode = http.StatusInternalServerError
		errorMsg = "An error occurred while creating the short URL"
	}
	
	w.WriteHeader(statusCode)
	if err := pages.CreateWithError(errorMsg).Render(r.Context(), w); err != nil {
		w.Write([]byte("Error rendering page"))
	}
}

// Dashboard renders the URL management dashboard page.
//
// GET /dashboard?limit=20&offset=0
//
// Query parameters:
//   - limit: maximum number of URLs to return (default: 20, max: 100)
//   - offset: number of URLs to skip (default: 0)
//
// Returns:
//   - 200: Successfully rendered dashboard
//   - 400: Invalid pagination parameters
//   - 500: Failed to load URLs
//
// Note: Currently uses a fixed user ID "authenticated-user".
// In production, this should come from session authentication.
func (h *PageHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	// Parse query parameters for pagination
	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)
	
	// Validate pagination parameters
	if limit < 0 || offset < 0 || limit > 100 {
		w.WriteHeader(http.StatusBadRequest)
		if err := pages.DashboardWithError("Invalid pagination parameters").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	// For now, use a fixed user ID. In a real app, this would come from a session cookie.
	// This matches the pattern used in the create page where the user provides an auth token.
	userID := "authenticated-user"
	
	// Fetch URLs using the list use case
	resp, err := h.listUseCase.Execute(r.Context(), application.ListURLsRequest{
		CreatedBy: userID,
		Limit:     limit,
		Offset:    offset,
	})
	
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := pages.DashboardWithError("Failed to load URLs").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	// Fetch click counts for each URL
	clickCounts := make(map[string]int64)
	for _, urlItem := range resp.URLs {
		// Find the URL by short code to get its ID
		urlEntity, err := h.urlRepo.FindByShortCode(r.Context(), urlItem.ShortCode)
		if err != nil {
			// If we can't find the URL, skip it (shouldn't happen but be defensive)
			continue
		}
		
		// Get the click count for this URL
		count, err := h.clickRepo.GetTotalClickCount(r.Context(), urlEntity.ID)
		if err != nil {
			// If we can't get the count, default to 0
			count = 0
		}
		clickCounts[urlItem.ShortCode] = count
	}
	
	// Render the dashboard
	w.WriteHeader(http.StatusOK)
	if err := pages.Dashboard(resp.URLs, clickCounts, resp.Total, limit, offset).Render(r.Context(), w); err != nil {
		w.Write([]byte("Error rendering page"))
	}
}

