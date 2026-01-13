package tailscale_test

import (
	"testing"

	// These imports verify that the Tailscale dependencies are available.
	// The test will fail to compile if the dependencies are not installed.
	_ "tailscale.com/client/tailscale"
	_ "tailscale.com/tsnet"
)

func TestTailscaleDependenciesAvailable(t *testing.T) {
	// This test verifies that the Tailscale packages are importable.
	// If this test compiles and runs, the dependencies are correctly installed.
	t.Log("Tailscale dependencies (tsnet, client/tailscale) are available")
}
