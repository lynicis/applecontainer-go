package applecontainer

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/lynicis/applecontainer-go/log"
	"github.com/lynicis/applecontainer-go/wait"
)

func TestCombineLifecycleHooks(t *testing.T) {
	var order []string

	makeReqHook := func(name string) ContainerRequestHook {
		return func(ctx context.Context, req *ContainerRequest) error {
			order = append(order, name)
			return nil
		}
	}

	makeCtrHook := func(name string) ContainerHook {
		return func(ctx context.Context, c Container) error {
			order = append(order, name)
			return nil
		}
	}

	defaults := ContainerLifecycleHooks{
		PreCreates:  []ContainerRequestHook{makeReqHook("default-pre")},
		PostCreates: []ContainerRequestHook{makeReqHook("default-post")},
	}

	user := ContainerLifecycleHooks{
		PreCreates:  []ContainerRequestHook{makeReqHook("user-pre")},
		PostCreates: []ContainerRequestHook{makeReqHook("user-post")},
		PreStarts:   []ContainerHook{makeCtrHook("user-start")},
	}

	combined := combineContainerHooks(defaults, user)

	// Execute PreCreates: should be default then user
	for _, h := range combined.PreCreates {
		_ = h(context.Background(), nil)
	}
	if len(order) != 2 || order[0] != "default-pre" || order[1] != "user-pre" {
		t.Errorf("unexpected PreCreates execution order: %v", order)
	}

	// Reset and execute PostCreates: should be user then default
	order = nil
	for _, h := range combined.PostCreates {
		_ = h(context.Background(), nil)
	}
	if len(order) != 2 || order[0] != "user-post" || order[1] != "default-post" {
		t.Errorf("unexpected PostCreates execution order: %v", order)
	}

	// Execute PreStarts: only user
	order = nil
	for _, h := range combined.PreStarts {
		_ = h(context.Background(), nil)
	}
	if len(order) != 1 || order[0] != "user-start" {
		t.Errorf("unexpected PreStarts execution: %v", order)
	}
}

type mockWaitStrategy struct {
	waited bool
}

func (m *mockWaitStrategy) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	m.waited = true
	return nil
}

func TestExecuteLifecycle(t *testing.T) {
	// 1. Create a temporary host file for the copy test
	tmpFile, err := os.CreateTemp("", "applecontainer-lifecycle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte("hello world")); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	var createCalled, startCalled bool
	var copiedData []byte
	var copiedPath string

	runner := &fakeRunner{
		runFn: func(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
			if len(args) > 0 {
				if args[0] == "create" {
					createCalled = true
					// Find the cidfile and write a dummy container ID
					var cidFile string
					for i, arg := range args {
						if arg == "--cidfile" && i+1 < len(args) {
							cidFile = args[i+1]
							break
						}
					}
					if cidFile != "" {
						err := os.WriteFile(cidFile, []byte("fake-lifecycle-cid"), 0644)
						if err != nil {
							return nil, nil, 1, err
						}
					}
				} else if args[0] == "start" {
					startCalled = true
				} else if args[0] == "cp" {
					// cp <hostPath> <containerPath>
					if len(args) >= 3 {
						copiedPath = args[2]
						if strings.HasPrefix(copiedPath, "fake-lifecycle-cid:") {
							copiedPath = strings.TrimPrefix(copiedPath, "fake-lifecycle-cid:")
						}
						hostPath := args[1]
						data, err := os.ReadFile(hostPath)
						if err == nil {
							copiedData = data
						}
					}
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
		log:      log.TestLogger(t),
		req: ContainerRequest{
			Image: "nginx",
			Files: []ContainerFile{
				{
					HostFilePath:      tmpFile.Name(),
					ContainerFilePath: "/app/hello.txt",
					FileMode:          0644,
				},
			},
			WaitingFor: &mockWaitStrategy{},
		},
	}

	// Set combined lifecycle hooks (combines defaults and user)
	c.lifecycle = []ContainerLifecycleHooks{
		defaultHooks(&c.req, c),
	}

	if c.id != "" {
		t.Fatal("expected container ID to be empty initially")
	}

	// Run lifecycle start
	err = c.executeLifecycle(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error running lifecycle: %v", err)
	}

	if !createCalled {
		t.Error("expected provider.CreateContainer to be called")
	}

	if c.id != "fake-lifecycle-cid" {
		t.Errorf("expected container ID 'fake-lifecycle-cid', got %q", c.id)
	}

	if !startCalled {
		t.Error("expected provider.StartContainer to be called")
	}

	if string(copiedData) != "hello world" || copiedPath != "/app/hello.txt" {
		t.Errorf("expected file 'hello world' to be copied to '/app/hello.txt', got content %q and path %q", copiedData, copiedPath)
	}

	ws := c.req.WaitingFor.(*mockWaitStrategy)
	if !ws.waited {
		t.Error("expected wait strategy to be executed")
	}

	if !c.IsRunning() {
		t.Error("expected container state to be running after lifecycle")
	}

	// Run lifecycle stop/cleanup
	err = c.executeLifecycle(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}

	if c.IsRunning() {
		t.Error("expected container state to not be running after stop lifecycle")
	}
}

func TestExecuteLifecycle_PreCreateError(t *testing.T) {
	runner := &fakeRunner{}
	p := &cliProvider{
		runner: runner,
		cfg:    Config{},
		log:    log.TestLogger(t),
	}
	c := &cliContainer{
		provider: p,
		log:      log.TestLogger(t),
		req: ContainerRequest{
			Image: "nginx",
		},
	}

	errCreate := errors.New("creation failure")
	c.lifecycle = []ContainerLifecycleHooks{
		{
			PreCreates: []ContainerRequestHook{
				func(ctx context.Context, req *ContainerRequest) error {
					return errCreate
				},
			},
		},
	}

	err := c.executeLifecycle(context.Background(), true)
	if !errors.Is(err, errCreate) {
		t.Errorf("expected create hook error %v, got %v", errCreate, err)
	}
}
