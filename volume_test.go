package applecontainer

import (
	"context"
	"testing"
)

func TestNewVolume(t *testing.T) {
	var capturedArgs []string
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			return nil, nil, 0, nil
		},
	}
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = nil }()

	vol, err := NewVolume(context.Background(),
		WithVolumeNameOption("my-custom-vol"),
		WithVolumeLabels(map[string]string{"env": "test"}),
		WithVolumeSize("10GB"),
		WithVolumeOpt("type", "tmpfs"),
	)
	if err != nil {
		t.Fatalf("failed to create volume: %v", err)
	}

	if vol.Name() != "my-custom-vol" {
		t.Errorf("expected volume name 'my-custom-vol', got %q", vol.Name())
	}

	expected := []string{"volume", "create", "--size", "10GB", "--opt", "type=tmpfs", "--label", "env=test", "my-custom-vol"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(capturedArgs), capturedArgs)
	}
	for i, arg := range capturedArgs {
		if arg != expected[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, arg, expected[i])
		}
	}

	err = vol.Remove(context.Background())
	if err != nil {
		t.Fatalf("failed to remove: %v", err)
	}
	if len(capturedArgs) != 3 || capturedArgs[0] != "volume" || capturedArgs[1] != "delete" || capturedArgs[2] != "my-custom-vol" {
		t.Errorf("unexpected delete args: %v", capturedArgs)
	}
}
