package tailscale

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
	"github.com/rs/zerolog"
	"tailscale.com/client/tailscale"
)

// WhoIsClient implements middleware.WhoIsClient using a Tailscale local client.
type WhoIsClient struct {
	server *Server
	logger zerolog.Logger
}

// NewWhoIsClient creates a new WhoIs client backed by the given Tailscale server.
func NewWhoIsClient(server *Server, logger zerolog.Logger) *WhoIsClient {
	return &WhoIsClient{
		server: server,
		logger: logger.With().Str("component", "tailscale-whois").Logger(),
	}
}

// WhoIs looks up the Tailscale identity for the given remote address.
// The remoteAddr should be in the format "ip:port" (e.g., "100.64.0.1:12345").
func (c *WhoIsClient) WhoIs(ctx context.Context, remoteAddr string) (*middleware.TailscaleUserProfile, error) {
	tsServer, err := c.server.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get local client: %w", err)
	}

	lc, err := tsServer.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get tailscale client: %w", err)
	}

	// Parse the remote address to get just the IP
	addrPort, err := netip.ParseAddrPort(remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote address %q: %w", remoteAddr, err)
	}

	whois, err := lc.WhoIs(ctx, addrPort.Addr().String())
	if err != nil {
		return nil, fmt.Errorf("WhoIs lookup failed: %w", err)
	}

	profile := &middleware.TailscaleUserProfile{
		LoginName:   whois.UserProfile.LoginName,
		DisplayName: whois.UserProfile.DisplayName,
		NodeName:    whois.Node.ComputedName,
	}

	c.logger.Debug().
		Str("login", profile.LoginName).
		Str("display_name", profile.DisplayName).
		Str("node", profile.NodeName).
		Str("remote_addr", remoteAddr).
		Msg("WhoIs lookup successful")

	return profile, nil
}

// Ensure WhoIsClient implements middleware.WhoIsClient
var _ middleware.WhoIsClient = (*WhoIsClient)(nil)

// LocalClientForTesting returns the underlying tailscale client for testing.
// This is used by integration tests that need to create listeners.
func (c *WhoIsClient) LocalClientForTesting() (*tailscale.LocalClient, error) {
	tsServer, err := c.server.LocalClient()
	if err != nil {
		return nil, err
	}
	return tsServer.LocalClient()
}
