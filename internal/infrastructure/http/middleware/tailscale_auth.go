package middleware

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
)

const (
	// TailscaleUserKey is the context key for storing Tailscale user identity
	TailscaleUserKey contextKey = "tailscale_user"
)

// TailscaleUserProfile contains the authenticated user's identity from Tailscale WhoIs.
type TailscaleUserProfile struct {
	LoginName   string // User's login name (e.g., "alice@example.com")
	DisplayName string // User's display name (e.g., "Alice Smith")
	NodeName    string // Name of the connecting node (e.g., "alice-laptop")
}

// UserID returns a unique identifier for the user, using LoginName.
func (p *TailscaleUserProfile) UserID() string {
	return p.LoginName
}

// WhoIsClient is an interface for Tailscale WhoIs lookups.
// This allows the middleware to be tested with a mock client.
type WhoIsClient interface {
	WhoIs(ctx context.Context, remoteAddr string) (*TailscaleUserProfile, error)
}

// TailscaleAuth returns a middleware that authenticates requests using Tailscale WhoIs.
//
// It extracts the remote IP from the request, calls the WhoIs API to identify
// the connecting Tailscale user, and stores the user profile in the request context.
func TailscaleAuth(client WhoIsClient, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if client == nil {
				logger.Error().Msg("tailscale WhoIs client is nil")
				respondJSONError(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			remoteAddr := r.RemoteAddr
			profile, err := client.WhoIs(r.Context(), remoteAddr)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("remote_addr", remoteAddr).
					Msg("tailscale WhoIs lookup failed")
				respondJSONError(w, "Unauthorized: unable to verify Tailscale identity", http.StatusUnauthorized)
				return
			}

			// Validate that we have a valid user identity
			if profile == nil || profile.LoginName == "" {
				logger.Warn().
					Str("remote_addr", remoteAddr).
					Msg("tailscale WhoIs returned empty user identity")
				respondJSONError(w, "Unauthorized: invalid Tailscale identity", http.StatusUnauthorized)
				return
			}

			logger.Debug().
				Str("login", profile.LoginName).
				Str("name", profile.DisplayName).
				Str("node", profile.NodeName).
				Str("remote_addr", remoteAddr).
				Msg("tailscale user authenticated")

			// Store user profile in context
			ctx := context.WithValue(r.Context(), TailscaleUserKey, profile)

			// Also set the standard UserID key for compatibility with existing code
			ctx = context.WithValue(ctx, UserIDKey, profile.UserID())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTailscaleUser extracts the Tailscale user profile from the request context.
func GetTailscaleUser(ctx context.Context) (*TailscaleUserProfile, bool) {
	profile, ok := ctx.Value(TailscaleUserKey).(*TailscaleUserProfile)
	return profile, ok
}
