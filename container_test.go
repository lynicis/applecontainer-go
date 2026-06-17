package applecontainer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lynicis/applecontainer-go/log"
)

func TestContainerEndpointMath_DirectIP(t *testing.T) {
	inspectJSON := `[
		{
			"id": "test-container-id",
			"status": {
				"networks": [
					{
						"network": "default",
						"ipv4Address": "192.168.64.9/24"
					}
				],
				"state": "running"
			}
		}
	]`

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) == 2 && args[0] == "inspect" && args[1] == "test-container-id" {
				return []byte(inspectJSON), nil, 0, nil
			}
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
		id:       "test-container-id",
		req: ContainerRequest{
			Image:           "nginx:latest",
			HostPortMapping: false,
		},
	}

	host, err := c.Host(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "192.168.64.9" {
		t.Errorf("expected host IP '192.168.64.9', got %q", host)
	}

	ip, err := c.ContainerIP(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "192.168.64.9" {
		t.Errorf("expected IP '192.168.64.9', got %q", ip)
	}

	mapped, err := c.MappedPort(context.Background(), "80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mapped != 80 {
		t.Errorf("expected mapped port 80, got %d", mapped)
	}

	mappedTcp, err := c.MappedPort(context.Background(), "80/tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mappedTcp != 80 {
		t.Errorf("expected mapped tcp port 80, got %d", mappedTcp)
	}

	endpoint, err := c.Endpoint(context.Background(), "80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint != "192.168.64.9:80" {
		t.Errorf("expected endpoint '192.168.64.9:80', got %q", endpoint)
	}

	portEndpoint, err := c.PortEndpoint(context.Background(), "80", "tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if portEndpoint != "192.168.64.9:80" {
		t.Errorf("expected portEndpoint '192.168.64.9:80', got %q", portEndpoint)
	}
}

func TestContainerEndpointMath_HostPortMapping(t *testing.T) {
	inspectJSON := `[
		{
			"id": "test-container-id",
			"configuration": {
				"publishedPorts": [
					{
						"containerPort": 80,
						"hostPort": 32768,
						"proto": "tcp"
					},
					{
						"containerPort": 443,
						"hostPort": 32769,
						"proto": "tcp"
					}
				]
			},
			"status": {
				"networks": [
					{
						"network": "default",
						"ipv4Address": "192.168.64.9/24"
					}
				],
				"state": "running"
			}
		}
	]`

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) == 2 && args[0] == "inspect" && args[1] == "test-container-id" {
				return []byte(inspectJSON), nil, 0, nil
			}
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
		id:       "test-container-id",
		req: ContainerRequest{
			Image:           "nginx:latest",
			HostPortMapping: true,
		},
	}

	host, err := c.Host(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "localhost" {
		t.Errorf("expected host 'localhost', got %q", host)
	}

	ip, err := c.ContainerIP(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "192.168.64.9" {
		t.Errorf("expected IP '192.168.64.9', got %q", ip)
	}

	mapped, err := c.MappedPort(context.Background(), "80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mapped != 32768 {
		t.Errorf("expected mapped port 32768, got %d", mapped)
	}

	mappedTcp, err := c.MappedPort(context.Background(), "443/tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mappedTcp != 32769 {
		t.Errorf("expected mapped tcp port 32769, got %d", mappedTcp)
	}

	// Unmapped port
	_, err = c.MappedPort(context.Background(), "8080")
	if err == nil {
		t.Fatal("expected error for unmapped port")
	}
	if !strings.Contains(err.Error(), "not mapped") {
		t.Errorf("expected error message to mention 'not mapped', got %v", err)
	}

	endpoint, err := c.Endpoint(context.Background(), "80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint != "localhost:32768" {
		t.Errorf("expected endpoint 'localhost:32768', got %q", endpoint)
	}
}

func TestContainerStartStopLifecycle(t *testing.T) {
	var startArgs []string
	var stopArgs []string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) > 0 && args[0] == "start" {
				startArgs = args
			} else if len(args) > 0 && args[0] == "stop" {
				stopArgs = args
			}
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
		id:       "test-lifecycle-container",
	}

	if c.IsRunning() {
		t.Fatal("expected container not to be running initially")
	}

	// Test Start
	err := c.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected Start error: %v", err)
	}

	if !c.IsRunning() {
		t.Fatal("expected container to be running after Start")
	}

	if len(startArgs) != 2 || startArgs[0] != "start" || startArgs[1] != "test-lifecycle-container" {
		t.Errorf("unexpected start args: %v", startArgs)
	}

	// Test Stop
	timeout := 10 * time.Second
	err = c.Stop(context.Background(), &timeout)
	if err != nil {
		t.Fatalf("unexpected Stop error: %v", err)
	}

	if c.IsRunning() {
		t.Fatal("expected container not to be running after Stop")
	}

	if len(stopArgs) != 4 || stopArgs[0] != "stop" || stopArgs[1] != "--time" || stopArgs[2] != "10" || stopArgs[3] != "test-lifecycle-container" {
		t.Errorf("unexpected stop args: %v", stopArgs)
	}
}

func TestContainerTerminateIdempotency(t *testing.T) {
	// 1. Success case
	{
		var stopCalled, deleteCalled bool
		runner := &fakeRunner{
			runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
				if len(args) > 0 && args[0] == "stop" {
					stopCalled = true
				}
				if len(args) > 0 && args[0] == "delete" {
					deleteCalled = true
				}
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
			id:       "test-idempotent-container",
		}
		c.isRunning.Store(true)

		err := c.Terminate(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !stopCalled || !deleteCalled {
			t.Errorf("expected both stop and delete to be called, got stop=%v delete=%v", stopCalled, deleteCalled)
		}
		if c.IsRunning() {
			t.Error("expected container to not be running after Terminate")
		}
	}

	// 2. Not found / already terminated case (idempotency check)
	{
		runner := &fakeRunner{
			runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
				if len(args) > 0 && (args[0] == "stop" || args[0] == "delete") {
					// Return exit error indicating container does not exist
					errStr := "container test-idempotent-container does not exist"
					return nil, []byte(errStr), 1, &runError{
						bin:    "container",
						args:   args,
						code:   1,
						stderr: errStr,
						cause:  errors.New("exit status 1"),
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

		c := &cliContainer{
			provider: p,
			id:       "test-idempotent-container",
		}
		c.isRunning.Store(true)

		// Calling Terminate should swallow the "does not exist" errors and return nil.
		err := c.Terminate(context.Background())
		if err != nil {
			t.Fatalf("expected Terminate to succeed idempotently, got error: %v", err)
		}
		if c.IsRunning() {
			t.Error("expected container to not be running after Terminate")
		}
	}

	// 3. Other error case (should propagate)
	{
		runner := &fakeRunner{
			runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
				if len(args) > 0 && args[0] == "delete" {
					return nil, []byte("permission denied"), 1, &runError{
						bin:    "container",
						args:   args,
						code:   1,
						stderr: "permission denied",
						cause:  errors.New("exit status 1"),
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

		c := &cliContainer{
			provider: p,
			id:       "test-idempotent-container",
		}

		err := c.Terminate(context.Background())
		if err == nil {
			t.Fatal("expected error to be propagated")
		}
		if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("expected error message to contain 'permission denied', got %v", err)
		}
	}
}
