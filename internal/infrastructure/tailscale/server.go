package tailscale

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/rs/zerolog"
	"tailscale.com/tsnet"
)

var (
	// ErrNilConfig is returned when a nil config is provided.
	ErrNilConfig = errors.New("config is nil")
	// ErrTailscaleDisabled is returned when Tailscale is not enabled in config.
	ErrTailscaleDisabled = errors.New("tailscale is not enabled")
	// ErrMissingHostname is returned when TailscaleHostname is empty.
	ErrMissingHostname = errors.New("tailscale hostname is required")
	// ErrMissingStateDir is returned when TailscaleStateDir is empty.
	ErrMissingStateDir = errors.New("tailscale state directory is required")
)

// Server wraps a tsnet.Server to provide Tailscale network functionality.
type Server struct {
	server   *tsnet.Server
	logger   zerolog.Logger
	hostname string

	mu      sync.Mutex
	closed  bool
	started bool
}

// NewServer creates a new Tailscale server wrapper configured from the application config.
// Returns an error if Tailscale is disabled or required configuration is missing.
func NewServer(cfg *config.Config, logger zerolog.Logger) (*Server, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}
	if !cfg.TailscaleEnabled {
		return nil, ErrTailscaleDisabled
	}
	if cfg.TailscaleHostname == "" {
		return nil, ErrMissingHostname
	}
	if cfg.TailscaleStateDir == "" {
		return nil, ErrMissingStateDir
	}

	tsServer := &tsnet.Server{
		Hostname: cfg.TailscaleHostname,
		Dir:      cfg.TailscaleStateDir,
	}

	// Set optional auth key for automated authentication
	if cfg.TailscaleAuthKey != "" {
		tsServer.AuthKey = cfg.TailscaleAuthKey
	}

	// Set optional control URL for headscale or custom control planes
	if cfg.TailscaleControlURL != "" {
		tsServer.ControlURL = cfg.TailscaleControlURL
	}

	logger.Info().
		Str("hostname", cfg.TailscaleHostname).
		Str("state_dir", cfg.TailscaleStateDir).
		Str("control_url", cfg.TailscaleControlURL).
		Msg("tailscale server configured")

	return &Server{
		server:   tsServer,
		logger:   logger.With().Str("component", "tailscale").Logger(),
		hostname: cfg.TailscaleHostname,
	}, nil
}

// Listen returns a net.Listener for the given network and address on the Tailscale network.
// This starts the Tailscale node if it hasn't been started yet.
func (s *Server) Listen(network, addr string) (net.Listener, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errors.New("server is closed")
	}
	// Mark as started before releasing lock to prevent race with Close()
	s.started = true
	s.mu.Unlock()

	s.logger.Info().
		Str("network", network).
		Str("addr", addr).
		Msg("starting tailscale listener")

	ln, err := s.server.Listen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on tailscale: %w", err)
	}

	s.logger.Info().
		Str("network", network).
		Str("addr", addr).
		Msg("tailscale listener started")

	return ln, nil
}

// Hostname returns the configured Tailscale hostname.
func (s *Server) Hostname() string {
	return s.hostname
}

// TSNetServer returns the underlying tsnet.Server for advanced operations.
// Callers can use this to get a LocalClient via tsServer.LocalClient().
// Returns an error if the server has been closed.
func (s *Server) TSNetServer() (*tsnet.Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, errors.New("server is closed")
	}
	return s.server, nil
}

// Close shuts down the Tailscale server.
// It is safe to call multiple times.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	// Only close the tsnet server if it was actually started
	// (tsnet.Server.Close panics if called before any listener is created)
	if !s.started {
		s.logger.Debug().Msg("tailscale server never started, skipping close")
		return nil
	}

	s.logger.Info().Msg("shutting down tailscale server")

	if err := s.server.Close(); err != nil {
		return fmt.Errorf("failed to close tailscale server: %w", err)
	}

	s.logger.Info().Msg("tailscale server shut down")
	return nil
}
