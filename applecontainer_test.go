package applecontainer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lynicis/applecontainer-go/wait"
)

func TestSessionID_Stability(t *testing.T) {
	s1 := SessionID()
	s2 := SessionID()
	if s1 == "" {
		t.Fatal("SessionID is empty")
	}
	if s1 != s2 {
		t.Errorf("expected stable SessionID within process, got %q and %q", s1, s2)
	}
}

func TestGenericLabels(t *testing.T) {
	lbls := GenericLabels()
	if lbls["applecontainer"] != "true" {
		t.Errorf("expected applecontainer label true, got %q", lbls["applecontainer"])
	}
	if lbls["applecontainer.session"] != SessionID() {
		t.Errorf("expected session label %q, got %q", SessionID(), lbls["applecontainer.session"])
	}
}

func TestCleanupContainer_NilSafe(t *testing.T) {
	// Should not panic or error
	CleanupContainer(t, nil)
}

func TestRun_AppliesOptionsAndOrchestrates(t *testing.T) {
	Reset()

	var createCalled, startCalled bool

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) > 0 {
				if args[0] == "--version" {
					return []byte("container version 1.0.0 (build: release, commit: test)\n"), nil, 0, nil
				} else if args[0] == "system" && args[1] == "status" {
					return []byte("running\n"), nil, 0, nil
				} else if args[0] == "create" {
					createCalled = true
					// find cidfile and write mock ID
					var cidFile string
					for i, arg := range args {
						if arg == "--cidfile" && i+1 < len(args) {
							cidFile = args[i+1]
							break
						}
					}
					if cidFile != "" {
						_ = os.WriteFile(cidFile, []byte("fake-run-cid"), 0644)
					}
				} else if args[0] == "start" {
					startCalled = true
				}
			}
			return nil, nil, 0, nil
		},
	}

	// Set override runner
	providerRunnerOverride = runner
	defer func() {
		providerRunnerOverride = nil
	}()

	// Execute Run with custom options
	var customOptionApplied bool
	customOpt := ContainerCustomizer(func(req *ContainerRequest) error {
		customOptionApplied = true
		return nil
	})

	c, err := Run(context.Background(), "nginx", customOpt)
	if err != nil {
		t.Fatalf("unexpected error running container: %v", err)
	}

	if !customOptionApplied {
		t.Error("expected custom option to be applied")
	}

	if !createCalled {
		t.Error("expected create command to be called")
	}

	if !startCalled {
		t.Error("expected start command to be called")
	}

	if c.id != "fake-run-cid" {
		t.Errorf("expected container ID 'fake-run-cid', got %q", c.id)
	}

	if !c.IsRunning() {
		t.Error("expected container to be running")
	}
}

func TestApplecontainerOptionsProper(t *testing.T) {
	rs := randomString("prefix-")
	require.Greater(t, len(rs), len("prefix-"))
	assert.True(t, strings.HasPrefix(rs, "prefix-"))

	var gotArgs []string
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			gotArgs = args
			return nil, nil, 0, nil
		},
	}
	oldOverride := providerRunnerOverride
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = oldOverride }()

	err := Prune(context.Background())
	require.NoError(t, err)
	require.Len(t, gotArgs, 1)
	assert.Equal(t, "prune", gotArgs[0])
}

func TestRun_Errors(t *testing.T) {
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) > 0 && args[0] == "--version" {
				return []byte("container version 1.0.0 (build: release, commit: test)\n"), nil, 0, nil
			}
			return nil, nil, 0, nil
		},
	}
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = nil }()

	t.Run("CustomizerError", func(t *testing.T) {
		errCustomizer := func(req *ContainerRequest) error { return errors.New("custom error") }
		_, err := Run(context.Background(), "nginx", errCustomizer)
		assert.ErrorContains(t, err, "custom error")
	})

	t.Run("ValidationError", func(t *testing.T) {
		_, err := Run(context.Background(), "") // Empty image name triggers validation error
		assert.Error(t, err)
	})

	t.Run("CreateContainerError", func(t *testing.T) {
		failRunner := &fakeRunner{
			runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
				if len(args) > 0 {
					switch args[0] {
					case "--version":
						return []byte("container version 1.0.0\n"), nil, 0, nil
					case "create":
						return nil, nil, 1, errors.New("create error")
					}
				}
				return nil, nil, 0, nil
			},
		}
		providerRunnerOverride = failRunner
		_, err := Run(context.Background(), "nginx")
		assert.ErrorContains(t, err, "create error")
	})
	providerRunnerOverride = runner
}

func TestRun_BuildImage(t *testing.T) {
	var buildCalled bool
	var deleteCalled bool
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) > 0 {
				if args[0] == "--version" {
					return []byte("container version 1.0.0\n"), nil, 0, nil
				} else if args[0] == "build" {
					buildCalled = true
					return nil, nil, 0, nil
				} else if args[0] == "image" && args[1] == "delete" {
					deleteCalled = true
					return nil, nil, 0, nil
				} else if args[0] == "create" {
					for i, arg := range args {
						if arg == "--cidfile" {
							_ = os.WriteFile(args[i+1], []byte("fake-run-cid"), 0644)
							break
						}
					}
				}
			}
			return nil, nil, 0, nil
		},
	}
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = nil }()

	ctx := context.Background()
	c, err := Run(ctx, "", func(req *ContainerRequest) error {
		val := "argVal"
		req.FromContainerfile = FromContainerfile{
			Context:   ".",
			KeepImage: false,
			File:      "Dockerfile.test",
			BuildArgs: map[string]*string{"argKey": &val},
			Target:    "dev",
			NoCache:   true,
			Pull:      true,
			Platform:  "linux/arm64",
			Secrets:   map[string]string{"mysecret": "/tmp/secret"},
			// Test empty tags too
			Tags: nil,
		}
		return nil
	})
	require.NoError(t, err)
	assert.True(t, buildCalled)
	assert.NotEmpty(t, c.image)

	// Trigger cleanup
	for _, f := range c.req.Cleanups {
		_ = f(ctx)
	}
	assert.True(t, deleteCalled)
}

type mockWait struct {
	err error
}

func (m mockWait) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	return m.err
}

func TestRun_Coverage(t *testing.T) {
	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) > 0 && args[0] == "--version" {
				return []byte("container version 1.0.0\n"), nil, 0, nil
			}
			if len(args) > 0 && args[0] == "create" {
				for i, arg := range args {
					if arg == "--cidfile" {
						_ = os.WriteFile(args[i+1], []byte("fake-run-cid"), 0644)
						break
					}
				}
			}
			return nil, nil, 0, nil
		},
		startFn: func(ctx context.Context, args []string, stdin io.Reader) (*exec.Cmd, io.Reader, io.Reader, error) {
			return nil, strings.NewReader("log data"), strings.NewReader(""), nil
		},
	}
	providerRunnerOverride = runner
	defer func() { providerRunnerOverride = nil }()

	// Wait error
	_, err := Run(context.Background(), "nginx", WithWaitStrategy(mockWait{err: errors.New("wait failed")}))
	assert.ErrorContains(t, err, "wait failed")

	// Missing copy file
	_, err = Run(context.Background(), "nginx", WithFiles(ContainerFile{HostFilePath: "/does/not/exist"}))
	assert.ErrorContains(t, err, "failed to read host file")

	// Version check error
	badRunner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			return nil, nil, 1, errors.New("exec error")
		},
	}
	providerRunnerOverride = badRunner
	_, err = Run(context.Background(), "nginx")
	assert.ErrorContains(t, err, "exec error")
	providerRunnerOverride = runner

	// LogWriters
	buf := new(bytes.Buffer)
	_, err = Run(context.Background(), "nginx", WithLogWriters(buf))
	assert.NoError(t, err)
}
