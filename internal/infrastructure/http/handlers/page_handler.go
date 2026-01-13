package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/session"
)

// PageHandler handles HTML page rendering
type PageHandler struct {
	createUseCase CreateURLUseCase
	listUseCase   ListURLsUseCase
	authTokens    []string
	sessionStore  *session.Store
	secureCookies bool
}

// NewPageHandler creates a new PageHandler
func NewPageHandler(
	createUseCase CreateURLUseCase,
	listUseCase ListURLsUseCase,
	authTokens []string,
	sessionStore *session.Store,
	secureCookies bool,
) *PageHandler {
	return &PageHandler{
		createUseCase: createUseCase,
		listUseCase:   listUseCase,
		authTokens:    authTokens,
		sessionStore:  sessionStore,
		secureCookies: secureCookies,
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

	// Validate auth token using constant-time comparison to prevent timing attacks.
	// Avoid early-exit across tokens to reduce timing signal during rotations.
	match, configured := middleware.ValidateTokenConstantTime(authToken, h.authTokens)
	if !configured {
		w.WriteHeader(http.StatusInternalServerError)
		if err := pages.CreateWithError("Server authentication configuration error").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	if !match {
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
// Note: Requires session authentication via RequireSession middleware.
func (h *PageHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Parse query parameters for pagination
	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)

	// Validate pagination parameters
	if limit <= 0 || offset < 0 || limit > 100 {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := pages.DashboardWithError("Invalid pagination parameters").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Get user ID from context - check both Tailscale and session auth
	// TailscaleAuth middleware sets UserIDKey, SessionMiddleware sets SessionUserIDKey
	var userID string
	var ok bool

	// First check for Tailscale user (via GetUserID which reads UserIDKey)
	if userID, ok = middleware.GetUserID(r.Context()); !ok || userID == "" {
		// Fallback to session-based user ID
		userID, ok = middleware.GetSessionUserID(r.Context())
	}

	if !ok || userID == "" {
		// This shouldn't happen if auth middleware is applied
		w.WriteHeader(http.StatusUnauthorized)
		if err := pages.DashboardWithError("Authentication required").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Fetch URLs using the list use case (includes click counts)
	resp, err := h.listUseCase.Execute(r.Context(), application.ListURLsRequest{
		CreatedBy: userID,
		Limit:     limit,
		Offset:    offset,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := pages.DashboardWithError("Failed to load URLs").Render(r.Context(), w); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Extract click counts from response (already fetched by use case)
	clickCounts := make(map[string]int64)
	for _, urlItem := range resp.URLs {
		clickCounts[urlItem.ShortCode] = urlItem.ClickCount
	}

	// Render the dashboard
	tailscaleLogin := ""
	if tsUser, ok := middleware.GetTailscaleUser(r.Context()); ok && tsUser != nil {
		tailscaleLogin = tsUser.LoginName
	}

	w.WriteHeader(http.StatusOK)
	if err := pages.Dashboard(resp.URLs, clickCounts, resp.Total, limit, offset, tailscaleLogin, tailscaleLogin == "").Render(r.Context(), w); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("Error rendering page"))
	}
}

// Login handles GET and POST requests for the login page
func (h *PageHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Handle GET request - show login form
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		if err := pages.Login("").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Handle POST request - process login
	if r.Method == http.MethodPost {
		h.handleLogin(w, r)
		return
	}

	// Method not allowed
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// handleLogin processes the login form submission
func (h *PageHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := pages.Login("Invalid form data").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Extract form values
	authToken := strings.TrimSpace(r.FormValue("auth_token"))

	// Validate auth token
	if authToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := pages.Login("Authentication token is required").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Validate auth token using constant-time comparison to prevent timing attacks.
	// Avoid early-exit across tokens to reduce timing signal during rotations.
	match, configured := middleware.ValidateTokenConstantTime(authToken, h.authTokens)
	if !configured {
		w.WriteHeader(http.StatusInternalServerError)
		if err := pages.Login("Authentication is not properly configured").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}
	if !match {
		w.WriteHeader(http.StatusUnauthorized)
		if err := pages.Login("Invalid authentication token").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Invalidate any existing session to prevent session fixation attacks
	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err == nil && cookie.Value != "" {
		h.sessionStore.Delete(cookie.Value)
	}

	// Create session
	userID := "authenticated-user"
	sess, err := h.sessionStore.Create(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := pages.Login("Failed to create session").Render(r.Context(), w); err != nil {
			w.Write([]byte("Error rendering page"))
		}
		return
	}

	// Set session cookie (24 hours)
	middleware.SetSessionCookie(w, sess.ID, 24*60*60, h.secureCookies)

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Logout handles the logout process
func (h *PageHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err == nil && cookie.Value != "" {
		// Delete session from store
		h.sessionStore.Delete(cookie.Value)
	}

	// Clear session cookie
	middleware.ClearSessionCookie(w, h.secureCookies)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
