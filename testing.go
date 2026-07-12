package applecontainer

import (
	"context"
	"testing"
)

// SkipIfProviderNotHealthy checks the provider health and skips the test if not healthy.
func SkipIfProviderNotHealthy(t testing.TB) {
	t.Helper()
	provider := newProvider(Read())
	if _, err := checkVersion(context.Background(), provider.runner); err != nil {
		t.Skipf("Skipping test: container provider not healthy: %v", err)
	}
}

// CleanupNetwork registers network deletion in t.Cleanup.
func CleanupNetwork(t testing.TB, nw *Network) {
	t.Helper()
	if nw == nil {
		return
	}
	t.Cleanup(func() {
		if err := nw.Remove(context.Background()); err != nil {
			t.Logf("applecontainer: failed to remove network %q during cleanup: %v", nw.Name(), err)
		}
	})
}
