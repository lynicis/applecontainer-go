package applecontainer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	if ins.Status != "running" {
		t.Errorf("got status %q, want 'running'", ins.Status)
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
		defer func() { _ = rc.Close() }()

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

func TestExecContainer(t *testing.T) {
	fakeCID := "test-container-id"
	var capturedArgs []string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			return []byte("output line\n"), nil, 0, nil
		},
	}

	p := &cliProvider{
		runner: runner,
		cfg:    Config{},
		log:    log.TestLogger(t),
	}

	cmd := []string{"echo", "hello"}
	exitCode, output, err := p.ExecContainer(context.Background(), fakeCID, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("got exit code %d, want 0", exitCode)
	}

	if string(output) != "output line\n" {
		t.Errorf("got output %q", string(output))
	}

	expectedArgs := []string{"exec", fakeCID, "echo", "hello"}
	if len(capturedArgs) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %v", len(expectedArgs), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expectedArgs[i] {
			t.Errorf("got %q, want %q", v, expectedArgs[i])
		}
	}

	// Test with options
	userOpt := func(o *processOptions) {
		o.User = "root"
	}
	workdirOpt := func(o *processOptions) {
		o.WorkingDir = "/app"
	}
	envOpt := func(o *processOptions) {
		o.Env = []string{"FOO=bar"}
	}

	_, _, _ = p.ExecContainer(context.Background(), fakeCID, cmd, userOpt, workdirOpt, envOpt)
	expectedArgsWithOptions := []string{"exec", "--user", "root", "--workdir", "/app", "--env", "FOO=bar", fakeCID, "echo", "hello"}
	if len(capturedArgs) != len(expectedArgsWithOptions) {
		t.Fatalf("expected %d args, got %v", len(expectedArgsWithOptions), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expectedArgsWithOptions[i] {
			t.Errorf("got %q, want %q", v, expectedArgsWithOptions[i])
		}
	}
}

func TestCopyToContainer(t *testing.T) {
	fakeCID := "test-container-id"
	var capturedArgs []string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			// We can verify that the source file in args exists and has the expected content
			if len(args) == 3 && args[0] == "cp" {
				srcFile := args[1]
				data, err := os.ReadFile(srcFile)
				if err != nil {
					return nil, nil, -1, err
				}
				if string(data) != "fake file content" {
					return nil, nil, -1, fmt.Errorf("unexpected content: %q", string(data))
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

	err := p.CopyToContainer(context.Background(), fakeCID, "/app/config.json", []byte("fake file content"), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedArgs) != 3 {
		t.Fatalf("expected 3 args, got %v", capturedArgs)
	}

	if capturedArgs[0] != "cp" {
		t.Errorf("expected command 'cp', got %q", capturedArgs[0])
	}

	expectedDest := fakeCID + ":/app/config.json"
	if capturedArgs[2] != expectedDest {
		t.Errorf("expected destination %q, got %q", expectedDest, capturedArgs[2])
	}
}

func TestCopyFileFromContainer(t *testing.T) {
	fakeCID := "test-container-id"
	var capturedArgs []string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedArgs = args
			// Simulate container cp by writing fake content to the destination path
			if len(args) == 3 && args[0] == "cp" {
				destPath := args[2]
				if err := os.WriteFile(destPath, []byte("retrieved file content"), 0644); err != nil {
					return nil, nil, -1, err
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

	rc, err := p.CopyFileFromContainer(context.Background(), fakeCID, "/app/config.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read from reader: %v", err)
	}

	if string(data) != "retrieved file content" {
		t.Errorf("got content %q, want retrieved file content", string(data))
	}

	// Verify file is deleted on Close
	tempFileObj, ok := rc.(*tempFileReadCloser)
	if !ok {
		t.Fatal("expected returned reader to be *tempFileReadCloser")
	}
	tempFilePath := tempFileObj.Name()

	if _, err := os.Stat(tempFilePath); os.IsNotExist(err) {
		t.Error("expected temp file to exist before Close")
	}

	if err := rc.Close(); err != nil {
		t.Fatalf("failed to close reader: %v", err)
	}

	if _, err := os.Stat(tempFilePath); !os.IsNotExist(err) {
		t.Error("expected temp file to be deleted after Close")
	}

	if len(capturedArgs) != 3 {
		t.Fatalf("expected 3 args, got %v", capturedArgs)
	}
	if capturedArgs[0] != "cp" {
		t.Errorf("expected command 'cp', got %q", capturedArgs[0])
	}
	expectedSrc := fakeCID + ":/app/config.json"
	if capturedArgs[1] != expectedSrc {
		t.Errorf("expected source %q, got %q", expectedSrc, capturedArgs[1])
	}
}

func TestImagePull(t *testing.T) {
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

	err := p.ImagePull(context.Background(), "nginx:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedArgs := []string{"image", "pull", "--progress", "plain", "nginx:latest"}
	if len(capturedArgs) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %v", len(expectedArgs), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expectedArgs[i] {
			t.Errorf("got %q, want %q", v, expectedArgs[i])
		}
	}
}

func TestImageInspect(t *testing.T) {
	fakeRef := "nginx:latest"
	mockJSON := `[{"id": "sha256:nginx-id"}]`
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

	ii, err := p.ImageInspect(context.Background(), fakeRef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ii == nil || ii.ID != "sha256:nginx-id" {
		t.Errorf("expected ID 'sha256:nginx-id', got %v", ii)
	}

	expectedArgs := []string{"image", "inspect", fakeRef}
	if len(capturedArgs) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %v", len(expectedArgs), capturedArgs)
	}
	for i, v := range capturedArgs {
		if v != expectedArgs[i] {
			t.Errorf("got %q, want %q", v, expectedArgs[i])
		}
	}
}

func TestHealth(t *testing.T) {
	var capturedCommands [][]string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			capturedCommands = append(capturedCommands, args)
			if len(args) == 1 && args[0] == "--version" {
				return []byte("container version 1.0.0 (build: release, commit: unspeci)\n"), nil, 0, nil
			}
			if len(args) == 2 && args[0] == "system" && args[1] == "status" {
				return []byte("running"), nil, 0, nil
			}
			return nil, nil, 0, nil
		},
	}

	p := &cliProvider{
		runner: runner,
		cfg:    Config{},
		log:    log.TestLogger(t),
	}

	err := p.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedCommands) != 2 {
		t.Fatalf("expected 2 commands, got %v", capturedCommands)
	}

	if capturedCommands[0][0] != "--version" {
		t.Errorf("expected first command to be --version, got %v", capturedCommands[0])
	}

	if capturedCommands[1][0] != "system" || capturedCommands[1][1] != "status" {
		t.Errorf("expected second command to be system status, got %v", capturedCommands[1])
	}
}

func TestParseImageInspect(t *testing.T) {
	// 1. Array
	ii1, err := parseImageInspect([]byte(`[{"id": "sha256:123"}]`))
	require.NoError(t, err)
	assert.Equal(t, "sha256:123", ii1.ID)

	// 2. Object
	ii2, err := parseImageInspect([]byte(`{"id": "sha256:456"}`))
	require.NoError(t, err)
	assert.Equal(t, "sha256:456", ii2.ID)

	// 3. Invalid JSON
	_, err = parseImageInspect([]byte(`invalid`))
	assert.ErrorContains(t, err, "failed to parse image inspect JSON")
}

func TestProviderErrorCases(t *testing.T) {
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			return nil, nil, 1, errors.New("provider error")
		},
	}
	p := &cliProvider{
		runner: runner,
		log:    log.TestLogger(t),
	}

	_, err := p.CreateContainer(context.Background(), &ContainerRequest{})
	assert.ErrorContains(t, err, "provider error")

	err = p.StartContainer(context.Background(), &cliContainer{id: "123"})
	assert.ErrorContains(t, err, "provider error")

	_, err = p.InspectContainer(context.Background(), "123")
	assert.ErrorContains(t, err, "provider error")

	err = p.ImagePull(context.Background(), "img")
	assert.ErrorContains(t, err, "provider error")

	_, err = p.ImageInspect(context.Background(), "img")
	assert.ErrorContains(t, err, "provider error")
}

func TestCopyToContainerErrors(t *testing.T) {
	p := &cliProvider{runner: &fakeRunner{}}
	err := p.CopyToContainer(context.Background(), "", "/dest", []byte{}, 0644)
	assert.ErrorContains(t, err, "cannot copy to empty container ID")

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			return nil, nil, 1, errors.New("cp fail")
		},
	}
	p2 := &cliProvider{runner: runner}
	err = p2.CopyToContainer(context.Background(), "id", "/dest", []byte{}, 0644)
	assert.ErrorContains(t, err, "cp fail")
}
