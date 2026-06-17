package applecontainer

import (
	"context"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

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

func TestStopContainer(t *testing.T) {
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

	// Test nil timeout (should use 5)
	err := p.StopContainer(context.Background(), fakeCID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedArgsDefault := []string{"stop", "--time", "5", fakeCID}
	if len(capturedArgs) != len(expectedArgsDefault) {
		t.Fatalf("expected %d args, got %v", len(expectedArgsDefault), capturedArgs)
	}
	for i, arg := range capturedArgs {
		if arg != expectedArgsDefault[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, arg, expectedArgsDefault[i])
		}
	}

	// Test custom timeout (e.g., 12 seconds)
	dur := 12 * time.Second
	err = p.StopContainer(context.Background(), fakeCID, &dur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedArgsCustom := []string{"stop", "--time", "12", fakeCID}
	if len(capturedArgs) != len(expectedArgsCustom) {
		t.Fatalf("expected %d args, got %v", len(expectedArgsCustom), capturedArgs)
	}
	for i, arg := range capturedArgs {
		if arg != expectedArgsCustom[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, arg, expectedArgsCustom[i])
		}
	}
}

func TestKillContainer(t *testing.T) {
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

	// Test kill without signal
	err := p.KillContainer(context.Background(), fakeCID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected1 := []string{"kill", fakeCID}
	if len(capturedArgs) != len(expected1) {
		t.Fatalf("expected %d args, got %v", len(expected1), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expected1[i] {
			t.Errorf("got %q, want %q", v, expected1[i])
		}
	}

	// Test kill with signal
	err = p.KillContainer(context.Background(), fakeCID, "SIGUSR1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected2 := []string{"kill", "--signal", "SIGUSR1", fakeCID}
	if len(capturedArgs) != len(expected2) {
		t.Fatalf("expected %d args, got %v", len(expected2), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expected2[i] {
			t.Errorf("got %q, want %q", v, expected2[i])
		}
	}
}

func TestDeleteContainer(t *testing.T) {
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

	// Test delete without force
	err := p.DeleteContainer(context.Background(), fakeCID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected1 := []string{"delete", fakeCID}
	if len(capturedArgs) != len(expected1) {
		t.Fatalf("expected %d args, got %v", len(expected1), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expected1[i] {
			t.Errorf("got %q, want %q", v, expected1[i])
		}
	}

	// Test delete with force
	err = p.DeleteContainer(context.Background(), fakeCID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected2 := []string{"delete", "--force", fakeCID}
	if len(capturedArgs) != len(expected2) {
		t.Fatalf("expected %d args, got %v", len(expected2), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expected2[i] {
			t.Errorf("got %q, want %q", v, expected2[i])
		}
	}
}

func TestInspectContainer(t *testing.T) {
	fakeCID := "test-container-id"
	mockJSON := `[{"id": "test-container-id", "status": {"state": "running"}}]`
	var capturedArgs []string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			return []byte(mockJSON), nil, 0, nil
		},
	}

	p := &cliProvider{
		runner: runner,
		cfg:    Config{},
		log:    log.TestLogger(t),
	}

	ins, err := p.InspectContainer(context.Background(), fakeCID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ins == nil {
		t.Fatal("expected inspect result to be non-nil")
	}

	if ins.ID != fakeCID {
		t.Errorf("got ID %q, want %q", ins.ID, fakeCID)
	}

	if ins.State.Status != "running" {
		t.Errorf("got status %q, want 'running'", ins.State.Status)
	}

	if len(capturedArgs) != 2 || capturedArgs[0] != "inspect" || capturedArgs[1] != fakeCID {
		t.Errorf("unexpected args: %v", capturedArgs)
	}
}

func TestContainerLogs(t *testing.T) {
	fakeCID := "test-container-id"
	
	// Test follow = false
	{
		var capturedArgs []string
		runner := &fakeRunner{
			runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
				capturedArgs = args
				return []byte("log line 1\nlog line 2\n"), nil, 0, nil
			},
		}

		p := &cliProvider{
			runner: runner,
			cfg:    Config{},
			log:    log.TestLogger(t),
		}

		rc, err := p.ContainerLogs(context.Background(), fakeCID, false, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("failed to read logs: %v", err)
		}

		if string(data) != "log line 1\nlog line 2\n" {
			t.Errorf("got %q, want logs", string(data))
		}

		expectedArgs := []string{"logs", "-n", "10", fakeCID}
		if len(capturedArgs) != len(expectedArgs) {
			t.Fatalf("expected %d args, got %v", len(expectedArgs), capturedArgs)
		}
		for i, v := range capturedArgs {
			if v != expectedArgs[i] {
				t.Errorf("got %q, want %q", v, expectedArgs[i])
			}
		}
	}

	// Test follow = true
	{
		var capturedArgs []string
		pr, pw := io.Pipe()
		runner := &fakeRunner{
			startFn: func(ctx context.Context, args []string, stdin io.Reader) (*exec.Cmd, io.Reader, io.Reader, error) {
				capturedArgs = args
				cmd := exec.Command("sleep", "10")
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				return cmd, pr, nil, nil
			},
		}

		p := &cliProvider{
			runner: runner,
			cfg:    Config{},
			log:    log.TestLogger(t),
		}

		rc, err := p.ContainerLogs(context.Background(), fakeCID, true, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Write to pipe
		go func() {
			_, _ = pw.Write([]byte("stream line\n"))
		}()

		buf := make([]byte, 12)
		n, err := rc.Read(buf)
		if err != nil {
			t.Fatalf("failed to read from log stream: %v", err)
		}
		if string(buf[:n]) != "stream line\n" {
			t.Errorf("got stream %q", string(buf[:n]))
		}

		_ = rc.Close()
		_ = pw.Close()

		expectedArgs := []string{"logs", "-f", fakeCID}
		if len(capturedArgs) != len(expectedArgs) {
			t.Fatalf("expected %d args, got %v", len(expectedArgs), capturedArgs)
		}
		for i, v := range capturedArgs {
			if v != expectedArgs[i] {
				t.Errorf("got %q, want %q", v, expectedArgs[i])
			}
		}
	}
}
