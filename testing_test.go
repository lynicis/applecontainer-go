package applecontainer

import (
	"context"
	"fmt"
	"testing"
)

func TestTestingProper(t *testing.T) {
	CleanupNetwork(t, nil)

	nw := &cliNetwork{
		name: "test-nw",
		provider: &cliProvider{
			runner: &fakeRunner{
				runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
					return nil, nil, 0, nil
				},
			},
		},
	}
	CleanupNetwork(t, nw)
}

type mockTB struct {
	testing.TB
	skipMsg string
}

func (m *mockTB) Skipf(format string, args ...any) {
	m.skipMsg = fmt.Sprintf(format, args...)
}

func (m *mockTB) Helper() {}

func TestSkipIfProviderNotHealthy(t *testing.T) {
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			return nil, nil, 1, fmt.Errorf("fake error")
		},
	}
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = nil }()

	mtb := &mockTB{}
	SkipIfProviderNotHealthy(mtb)

	if mtb.skipMsg == "" {
		t.Errorf("expected skip message to be set")
	}
}
