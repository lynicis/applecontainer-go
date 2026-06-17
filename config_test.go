package applecontainer

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner is a test double for commandRunner. It is reused by later tests.
type fakeRunner struct {
	stdout    string
	stderr    string
	code      int
	err       error
	gotArgs   []string
	gotStdin  []byte
	callCount int
	runFn     func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error)
	startFn   func(ctx context.Context, args []string, stdin io.Reader) (*exec.Cmd, io.Reader, io.Reader, error)
}

func (f *fakeRunner) Run(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
	f.gotArgs = args
	f.gotStdin = stdin
	f.callCount++
	if f.runFn != nil {
		return f.runFn(ctx, args, stdin)
	}
	return []byte(f.stdout), []byte(f.stderr), f.code, f.err
}

func (f *fakeRunner) Start(ctx context.Context, args []string, stdin io.Reader) (*exec.Cmd, io.Reader, io.Reader, error) {
	if f.startFn != nil {
		return f.startFn(ctx, args, stdin)
	}
	return nil, nil, nil, errors.New("fakeRunner: Start not implemented")
}

func TestReadEnvOverridesProperties(t *testing.T) {
	Reset()
	dir := t.TempDir()
	propsPath := filepath.Join(dir, "applecontainer.properties")
	props := []byte("container.binary.path=/usr/bin/from-props\ncontainer.default.network=props-net\n")
	if err := os.WriteFile(propsPath, props, 0o600); err != nil {
		t.Fatal(err)
	}
	original := propertiesPath
	propertiesPath = propsPath
	t.Cleanup(func() { propertiesPath = original })

	t.Setenv("CONTAINER_BINARY", "/usr/bin/from-env")
	t.Setenv("CONTAINER_DEBUG", "true")

	cfg := Read()
	if cfg.BinaryPath != "/usr/bin/from-env" {
		t.Fatalf("BinaryPath=%q want /usr/bin/from-env (env should override properties)", cfg.BinaryPath)
	}
	if cfg.Debug != true {
		t.Fatalf("Debug=%v want true", cfg.Debug)
	}
	if cfg.DefaultNetwork != "props-net" {
		t.Fatalf("DefaultNetwork=%q want props-net (properties should be read)", cfg.DefaultNetwork)
	}
}

func TestVersionCheckParsesRealFormat(t *testing.T) {
	r := &fakeRunner{stdout: "container version 1.0.0 (build: release, commit: unspeci)\n"}
	got, err := checkVersion(context.Background(), r)
	if err != nil {
		t.Fatalf("checkVersion: %v", err)
	}
	if got != "1.0.0" {
		t.Fatalf("version=%q want 1.0.0", got)
	}
	if len(r.gotArgs) == 0 || r.gotArgs[0] != "--version" {
		t.Fatalf("args=%v want [--version]", r.gotArgs)
	}
}

func TestVersionCheckRejectsOldVersion(t *testing.T) {
	r := &fakeRunner{stdout: "container version 0.9.0 (build: release, commit: unspeci)\n"}
	_, err := checkVersion(context.Background(), r)
	if err == nil {
		t.Fatal("want error for version 0.9.0")
	}
	if !strings.Contains(err.Error(), "0.9.0") {
		t.Fatalf("error should mention the bad version: %v", err)
	}
}

func TestVersionCheckMissingBinaryHelpful(t *testing.T) {
	r := &fakeRunner{err: errors.New(`exec: "container": executable file not found in $PATH`), code: -1}
	_, err := checkVersion(context.Background(), r)
	if err == nil {
		t.Fatal("want error for missing binary")
	}
	if !strings.Contains(err.Error(), "container") {
		t.Fatalf("error should be helpful and mention container: %v", err)
	}
}
