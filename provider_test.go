package applecontainer

import (
	"context"
	"os"
	"testing"

	"github.com/lynicis/applecontainer-go/log"
)

func TestCreateContainer(t *testing.T) {
	fakeCID := "1234567890abcdef1234567890abcdef"
	var capturedArgs []string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			// Find the cidfile arg and write fakeCID to it
			for i, arg := range args {
				if arg == "--cidfile" && i+1 < len(args) {
					cidPath := args[i+1]
					if err := os.WriteFile(cidPath, []byte(fakeCID), 0644); err != nil {
						return nil, nil, -1, err
					}
					break
				}
			}
			return nil, nil, 0, nil
		},
	}

	p := &cliProvider{
		runner: runner,
		cfg:    Config{},
		log:    log.TestLogger(t),
	}

	req := &ContainerRequest{
		Image: "nginx:latest",
	}

	c, err := p.CreateContainer(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c == nil {
		t.Fatal("expected container to be non-nil")
	}

	if c.id != fakeCID {
		t.Errorf("expected container ID %q, got %q", fakeCID, c.id)
	}

	// Verify command and arguments
	if len(capturedArgs) < 4 {
		t.Fatalf("expected at least 4 arguments, got %v", capturedArgs)
	}
	if capturedArgs[0] != "create" {
		t.Errorf("expected command 'create', got %q", capturedArgs[0])
	}
	// Image should be the last argument
	lastArg := capturedArgs[len(capturedArgs)-1]
	if lastArg != "nginx:latest" {
		t.Errorf("expected last arg to be 'nginx:latest', got %q", lastArg)
	}
}

func TestStartContainer(t *testing.T) {
	fakeCID := "test-container-id"
	var capturedArgs []string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			return nil, nil, 0, nil
		},
	}

	p := &cliProvider{
		runner: runner,
		cfg:    Config{},
		log:    log.TestLogger(t),
	}

	c := &cliContainer{
		provider: p,
		id:       fakeCID,
	}

	err := p.StartContainer(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedArgs) != 2 {
		t.Fatalf("expected 2 arguments, got %v", capturedArgs)
	}
	if capturedArgs[0] != "start" {
		t.Errorf("expected 'start', got %q", capturedArgs[0])
	}
	if capturedArgs[1] != fakeCID {
		t.Errorf("expected container ID %q, got %q", fakeCID, capturedArgs[1])
	}
}
