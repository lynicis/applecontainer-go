package applecontainer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	nw, err := NewNetwork(context.Background(), NetworkRequest{
		Name:       "my-custom-net",
		Labels:     map[string]string{"env": "test"},
		Driver:     "bridge",
		Internal:   true,
		EnableIPv6: true,
		Subnet:     "10.0.0.0/24",
		SubnetV6:   "fd00::/64",
	})
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

func TestNetworkOptionsProper(t *testing.T) {
	name := generateNetworkName()
	assert.NotEmpty(t, name)

	req := &ContainerRequest{}
	nw := &cliNetwork{name: "fake"}

	customizer := WithNetwork([]string{"alias"}, nw)
	err := customizer(req)
	require.NoError(t, err)

	require.Len(t, req.Networks, 1)
	assert.Equal(t, "fake", req.Networks[0])

	require.Len(t, req.NetworkAliases["fake"], 1)
	assert.Equal(t, "alias", req.NetworkAliases["fake"][0])
}

func TestWithNewNetwork(t *testing.T) {
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			return nil, nil, 0, nil
		},
	}
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = nil }()

	req := &ContainerRequest{}
	customizer := WithNewNetwork(context.Background(), []string{"alias1"}, NetworkRequest{Name: "test-new-nw"})
	err := customizer(req)
	require.NoError(t, err)

	require.Len(t, req.Networks, 1)
	assert.Equal(t, "test-new-nw", req.Networks[0])
	require.Len(t, req.NetworkAliases["test-new-nw"], 1)
	assert.Equal(t, "alias1", req.NetworkAliases["test-new-nw"][0])
}
