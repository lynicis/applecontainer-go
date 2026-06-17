package applecontainer

import (
	"context"
	"testing"
)

func TestNewNetwork(t *testing.T) {
	var capturedArgs []string
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			return nil, nil, 0, nil
		},
	}
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = nil }()

	nw, err := NewNetwork(context.Background(),
		WithNetworkNameOption("my-custom-net"),
		WithNetworkLabels(map[string]string{"env": "test"}),
		WithNetworkDriver("bridge"),
		WithInternal(true),
		WithEnableIPv6(true),
		WithSubnet("10.0.0.0/24"),
		WithSubnetV6("fd00::/64"),
	)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}

	if nw.Name() != "my-custom-net" {
		t.Errorf("expected network name 'my-custom-net', got %q", nw.Name())
	}

	expected := []string{"network", "create", "--driver", "bridge", "--internal", "--ipv6", "--subnet", "10.0.0.0/24", "--subnet-v6", "fd00::/64", "--label", "env=test", "my-custom-net"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(capturedArgs), capturedArgs)
	}
	for i, arg := range capturedArgs {
		if arg != expected[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, arg, expected[i])
		}
	}

	err = nw.Remove(context.Background())
	if err != nil {
		t.Fatalf("failed to remove: %v", err)
	}
	if len(capturedArgs) != 3 || capturedArgs[0] != "network" || capturedArgs[1] != "delete" || capturedArgs[2] != "my-custom-net" {
		t.Errorf("unexpected delete args: %v", capturedArgs)
	}
}
