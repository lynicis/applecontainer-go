package applecontainer

import (
	"context"
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
