package tailscale_test

import (
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/tailscale"
	"github.com/rs/zerolog"
)

func TestNewServer_NilConfig(t *testing.T) {
	logger := zerolog.Nop()
	_, err := tailscale.NewServer(nil, logger)
	if err == nil {
		t.Fatal("Expected error for nil config, got nil")
	}
}

func TestNewServer_TailscaleDisabled(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled: false,
	}
	logger := zerolog.Nop()

	_, err := tailscale.NewServer(cfg, logger)
	if err == nil {
		t.Fatal("Expected error when Tailscale is disabled, got nil")
	}
}

func TestNewServer_MissingHostname(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "",
		TailscaleStateDir: "/tmp/ts-state",
	}
	logger := zerolog.Nop()

	_, err := tailscale.NewServer(cfg, logger)
	if err == nil {
		t.Fatal("Expected error for missing hostname, got nil")
	}
}

func TestNewServer_MissingStateDir(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "test-host",
		TailscaleStateDir: "",
	}
	logger := zerolog.Nop()

	_, err := tailscale.NewServer(cfg, logger)
	if err == nil {
		t.Fatal("Expected error for missing state dir, got nil")
	}
}

func TestNewServer_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "test-host",
		TailscaleStateDir: t.TempDir(),
	}
	logger := zerolog.Nop()

	server, err := tailscale.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if server == nil {
		t.Fatal("Expected server to be non-nil")
	}

	// Verify hostname is accessible
	if server.Hostname() != "test-host" {
		t.Errorf("Expected hostname 'test-host', got: %s", server.Hostname())
	}

	// Clean up
	if err := server.Close(); err != nil {
		t.Errorf("Error closing server: %v", err)
	}
}

func TestNewServer_WithAuthKey(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "test-host",
		TailscaleStateDir: t.TempDir(),
		TailscaleAuthKey:  "tskey-auth-test",
	}
	logger := zerolog.Nop()

	server, err := tailscale.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer server.Close()

	if server == nil {
		t.Fatal("Expected server to be non-nil")
	}
}

func TestNewServer_WithControlURL(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:    true,
		TailscaleHostname:   "test-host",
		TailscaleStateDir:   t.TempDir(),
		TailscaleControlURL: "https://headscale.example.com",
	}
	logger := zerolog.Nop()

	server, err := tailscale.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer server.Close()

	if server == nil {
		t.Fatal("Expected server to be non-nil")
	}
}

func TestServer_Close_Idempotent(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "test-host",
		TailscaleStateDir: t.TempDir(),
	}
	logger := zerolog.Nop()

	server, err := tailscale.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Close should be safe to call multiple times
	if err := server.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

func TestServer_TSNetServer_WhenOpen(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "test-host",
		TailscaleStateDir: t.TempDir(),
	}
	logger := zerolog.Nop()

	server, err := tailscale.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer server.Close()

	tsServer, err := server.TSNetServer()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if tsServer == nil {
		t.Fatal("Expected tsnet.Server to be non-nil")
	}
}

func TestServer_TSNetServer_WhenClosed(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "test-host",
		TailscaleStateDir: t.TempDir(),
	}
	logger := zerolog.Nop()

	server, err := tailscale.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Close the server first
	if err := server.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// TSNetServer should return an error when closed
	_, err = server.TSNetServer()
	if err == nil {
		t.Fatal("Expected error when server is closed, got nil")
	}
}

func TestServer_Listen_WhenClosed(t *testing.T) {
	cfg := &config.Config{
		TailscaleEnabled:  true,
		TailscaleHostname: "test-host",
		TailscaleStateDir: t.TempDir(),
	}
	logger := zerolog.Nop()

	server, err := tailscale.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Close the server first
	if err := server.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Listen should return an error when closed
	_, err = server.Listen("tcp", ":443")
	if err == nil {
		t.Fatal("Expected error when server is closed, got nil")
	}
}
