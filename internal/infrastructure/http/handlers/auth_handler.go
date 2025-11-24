package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/session"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	sessionRepo    session.Repository
	authToken      string
	sessionTimeout time.Duration
	secureCookies  bool
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(sessionRepo session.Repository, authToken string, sessionTimeout time.Duration, secureCookies bool) *AuthHandler {
	return &AuthHandler{
		sessionRepo:    sessionRepo,
		authToken:      authToken,
		sessionTimeout: sessionTimeout,
		secureCookies:  secureCookies,
	}
}

// LoginRequest represents the JSON request body for login
type LoginRequest struct {
	Token string `json:"token"`
}

// LoginResponse represents the JSON response for login
type LoginResponse struct {
	Message string `json:"message"`
}

// LogoutResponse represents the JSON response for logout
type LogoutResponse struct {
	Message string `json:"message"`
}

// Login handles POST /api/auth/login - Authenticate and create session
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate token is provided
	if req.Token == "" {
		respondError(w, "token is required", http.StatusBadRequest)
		return
	}

	// Validate token using constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(h.authToken)) != 1 {
		respondError(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Extract IP address from request
	ipAddress := h.extractIPAddress(r)

	// Extract User-Agent from request header
	userAgent := r.Header.Get("User-Agent")

	// Create new session
	// Using static user ID for now (future: JWT claims or user lookup)
	sess, err := session.NewSession("authenticated-user", ipAddress, userAgent)
	if err != nil {
		respondError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// Save session to repository
	if err := h.sessionRepo.Create(r.Context(), sess); err != nil {
		respondError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	middleware.SetSessionCookie(w, sess.ID, middleware.SessionCookieMaxAge, h.secureCookies)

	// Respond with success
	respondJSON(w, LoginResponse{
		Message: "Login successful",
	}, http.StatusOK)
}

// Logout handles POST /api/auth/logout - Destroy session
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session from context (set by session middleware)
	sess, ok := middleware.GetSession(r.Context())
	if !ok {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Delete session from repository
	if err := h.sessionRepo.Delete(r.Context(), sess.ID); err != nil {
		respondError(w, "failed to delete session", http.StatusInternalServerError)
		return
	}

	// Clear session cookie
	middleware.ClearSessionCookie(w)

	// Respond with success
	respondJSON(w, LogoutResponse{
		Message: "Logout successful",
	}, http.StatusOK)
}

// extractIPAddress extracts and cleans the IP address from the request
func (h *AuthHandler) extractIPAddress(r *http.Request) string {
	// Get IP from RemoteAddr
	ipAddress := r.RemoteAddr

	// Remove port if present (format is typically "IP:port")
	if idx := strings.LastIndex(ipAddress, ":"); idx != -1 {
		ipAddress = ipAddress[:idx]
	}

	// Remove brackets from IPv6 addresses
	ipAddress = strings.Trim(ipAddress, "[]")

	return ipAddress
}
