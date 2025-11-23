package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")

	// Encode to a buffer first to catch errors before writing status code
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		// Log error and send internal server error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to encode response"}`))
		return
	}

	w.WriteHeader(statusCode)
	w.Write(buf.Bytes())
}

// respondError writes a JSON error response
func respondError(w http.ResponseWriter, message string, statusCode int) {
	respondJSON(w, ErrorResponse{Error: message}, statusCode)
}

// handleDomainError maps domain errors to HTTP status codes
// This is a shared helper function used across handlers to maintain consistent error responses
func handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, url.ErrURLNotFound):
		respondError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, url.ErrDuplicateShortCode):
		respondError(w, err.Error(), http.StatusConflict)
	case errors.Is(err, url.ErrInvalidShortCode):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrEmptyShortCode):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrInvalidOriginalURL):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrEmptyOriginalURL):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrInvalidCreatedBy):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrUnauthorizedDeletion):
		respondError(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, url.ErrMissingURLScheme):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrInvalidURLScheme):
		respondError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, url.ErrMissingURLHost):
		respondError(w, err.Error(), http.StatusBadRequest)
	default:
		respondError(w, "internal server error", http.StatusInternalServerError)
	}
}

// parseQueryInt parses an integer query parameter with a default value
func parseQueryInt(r *http.Request, key string, defaultValue int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
