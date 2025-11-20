package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// RedirectUseCase defines the interface for redirect operations
type RedirectUseCase interface {
	Execute(ctx context.Context, req application.RedirectRequest) (*application.RedirectResponse, error)
}

// RedirectHandler handles HTTP redirect requests
type RedirectHandler struct {
	redirectUseCase RedirectUseCase
}

// NewRedirectHandler creates a new RedirectHandler
func NewRedirectHandler(redirectUseCase RedirectUseCase) *RedirectHandler {
	return &RedirectHandler{
		redirectUseCase: redirectUseCase,
	}
}

// Redirect handles GET /:shortCode - Redirect to original URL
func (h *RedirectHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	// Extract short code from URL path
	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		http.NotFound(w, r)
		return
	}

	// Extract analytics data from request
	// Note: IP address extraction is intentionally deferred until GeoIP integration
	referrer := r.Header.Get("Referer")
	userAgent := r.Header.Get("User-Agent")
	country := "" // Country is empty for now (GeoIP not implemented yet)

	// Execute redirect use case
	resp, err := h.redirectUseCase.Execute(r.Context(), application.RedirectRequest{
		ShortCode: shortCode,
		Referrer:  referrer,
		UserAgent: userAgent,
		Country:   country,
	})

	if err != nil {
		handleRedirectError(w, r, err)
		return
	}

	// Redirect to original URL with 302 status code
	http.Redirect(w, r, resp.OriginalURL, http.StatusFound)
}

// handleRedirectError maps redirect errors to HTTP responses
func handleRedirectError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, url.ErrURLNotFound) {
		// Render HTML 404 page for not found errors
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if renderErr := pages.NotFound().Render(r.Context(), w); renderErr != nil {
			// Fallback to plain text if template rendering fails
			http.Error(w, "Not Found", http.StatusNotFound)
		}
		return
	}
	// For other errors, render HTML 500 page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	if renderErr := pages.InternalError("An error occurred while processing your request").Render(r.Context(), w); renderErr != nil {
		// Fallback to plain text if template rendering fails
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
