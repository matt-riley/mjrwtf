package handlers

import (
	"net/http"

	"github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"
)

// PageHandler handles HTML page rendering
type PageHandler struct{}

// NewPageHandler creates a new PageHandler
func NewPageHandler() *PageHandler {
	return &PageHandler{}
}

// Home renders the home page
func (h *PageHandler) Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Home().Render(r.Context(), w); err != nil {
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// NotFound renders the 404 error page
func (h *PageHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := pages.NotFound().Render(r.Context(), w); err != nil {
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// InternalError renders the 500 error page
func (h *PageHandler) InternalError(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	if err := pages.InternalError(message).Render(r.Context(), w); err != nil {
		// Fallback to plain text if template fails
		http.Error(w, message, http.StatusInternalServerError)
	}
}
