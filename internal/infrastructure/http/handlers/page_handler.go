package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"
	"github.com/matt-riley/mjrwtf/internal/application"
)

// PageHandler handles HTML page rendering
type PageHandler struct {
	createUseCase CreateURLUseCase
}

// NewPageHandler creates a new PageHandler
func NewPageHandler(createUseCase CreateURLUseCase) *PageHandler {
	return &PageHandler{
		createUseCase: createUseCase,
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
	
	// Create context with user ID based on auth token
	// For now, use a static user ID since we don't have JWT parsing
	userID := "authenticated-user"
	ctx := context.WithValue(r.Context(), "userID", userID)
	
	// Call the create URL use case
	resp, err := h.createUseCase.Execute(ctx, application.CreateURLRequest{
		OriginalURL: originalURL,
		CreatedBy:   userID,
	})
	
	if err != nil {
		// Extract user-friendly error message
		errorMsg := err.Error()
		// Clean up technical error prefixes
		if strings.Contains(errorMsg, "failed to create shortened URL:") {
			errorMsg = strings.TrimPrefix(errorMsg, "failed to create shortened URL: ")
		}
		
		w.WriteHeader(http.StatusBadRequest)
		if err := pages.CreateWithError(errorMsg).Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	
	// Render success page with result
	w.WriteHeader(http.StatusOK)
	if err := pages.CreateWithResult(resp.ShortCode, resp.ShortURL, resp.OriginalURL).Render(r.Context(), w); err != nil {
		w.Write([]byte("Error rendering page"))
	}
}
