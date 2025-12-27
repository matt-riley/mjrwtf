package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

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

const maxJSONBodyBytes int64 = 1 << 20 // 1MB

type jsonUnknownFieldError struct {
	Field string
}

func (e *jsonUnknownFieldError) Error() string {
	if e.Field == "" {
		return "unknown field"
	}
	return "unknown field: " + e.Field
}

// decodeJSONBody decodes a JSON request body with strict validation including size limits,
// unknown field rejection, and single-value enforcement.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		// encoding/json returns an *errors.errorString for unknown fields, so normalize it
		// into a typed error we can match on without depending on string checks elsewhere.
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			field := strings.TrimPrefix(err.Error(), "json: unknown field ")
			field = strings.Trim(field, "\"")
			return &jsonUnknownFieldError{Field: field}
		}
		return err
	}

	// Ensure there's only one JSON value.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON value")
	}

	return nil
}

// respondJSONDecodeError writes an appropriate error response based on the JSON decoding error type.
func respondJSONDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		respondError(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	var unknownFieldErr *jsonUnknownFieldError
	if errors.As(err, &unknownFieldErr) {
		respondError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		respondError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var unmarshalTypeErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeErr) {
		respondError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if errors.Is(err, io.ErrUnexpectedEOF) {
		respondError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	respondError(w, "invalid JSON", http.StatusBadRequest)
}
