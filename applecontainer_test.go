package applecontainer

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type cleanupTrackingContainer struct {
	Container
	terminated bool
}

func (c *cleanupTrackingContainer) Terminate(ctx context.Context) error {
	c.terminated = true
	return nil
}

func TestCleanupContainer_RegistersCleanup(t *testing.T) {
	tc := &cleanupTrackingContainer{}

	// Create sub-test to trigger cleanup
	t.Run("SubTest", func(subT *testing.T) {
		CleanupContainer(subT, tc)
	})

	if !tc.terminated {
		t.Error("expected container to be terminated after sub-test cleanup")
	}
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
