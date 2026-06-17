package applecontainer

import (
	"context"
	"testing"
)

func TestDefaultBuildHook(t *testing.T) {
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
	}

	c := &cliContainer{
		provider: p,
	}

	req := &ContainerRequest{
		FromContainerfile: FromContainerfile{
			Context:   ".",
			File:      "Dockerfile.test",
			KeepImage: false,
			BuildArgs: map[string]*string{
				"ARG1": pointerToString("val1"),
			},
			Target:   "builder",
			NoCache:  true,
			Pull:     true,
			Platform: "linux/amd64",
			Secrets: map[string]string{
				"mysec": "/tmp/sec",
			},
		},
	}

	c.lifecycle = []ContainerLifecycleHooks{{}}

	err := defaultBuildHook(context.Background(), req, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Image == "" {
		t.Error("expected req.Image to be set to generated tag")
	}

	if c.image != req.Image {
		t.Errorf("expected container image to match tag: got %q, want %q", c.image, req.Image)
	}

	expectedPrefix := []string{"build", "-t", req.Image, "--progress", "plain", "-f", "Dockerfile.test"}
	for i, v := range expectedPrefix {
		if capturedArgs[i] != v {
			t.Errorf("arg[%d]: got %q, want %q", i, capturedArgs[i], v)
		}
	}

	hasArg := func(flag, val string) bool {
		for i := 0; i < len(capturedArgs)-1; i++ {
			if capturedArgs[i] == flag && capturedArgs[i+1] == val {
				return true
			}
		}
		return false
	}
	hasFlag := func(flag string) bool {
		for _, v := range capturedArgs {
			if v == flag {
				return true
			}
		}
		return false
	}

	if !hasArg("--build-arg", "ARG1=val1") {
		t.Error("missing --build-arg ARG1=val1")
	}
	if !hasArg("--target", "builder") {
		t.Error("missing --target builder")
	}
	if !hasFlag("--no-cache") {
		t.Error("missing --no-cache")
	}
	if !hasFlag("--pull") {
		t.Error("missing --pull")
	}
	if !hasArg("--platform", "linux/amd64") {
		t.Error("missing --platform linux/amd64")
	}
	if !hasArg("--secret", "id=mysec,src=/tmp/sec") {
		t.Error("missing --secret id=mysec,src=/tmp/sec")
	}
	if capturedArgs[len(capturedArgs)-1] != "." {
		t.Errorf("expected last arg to be '.', got %q", capturedArgs[len(capturedArgs)-1])
	}

	if len(c.lifecycle[0].PostTerminates) != 1 {
		t.Error("expected 1 PostTerminates hook for deleting image")
	}
}

func pointerToString(s string) *string {
	return &s
}
