package applecontainer

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecRunnerReturnsStdout(t *testing.T) {
	r := newExecRunner("echo")
	out, stderr, code, err := r.Run(context.Background(), []string{"hi"}, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if string(out) != "hi\n" {
		t.Fatalf("out=%q", out)
	}
}

func TestExecRunnerPassesStdin(t *testing.T) {
	r := newExecRunner("cat")
	out, _, code, err := r.Run(context.Background(), nil, []byte("ping"))
	if err != nil || code != 0 {
		t.Fatalf("err=%v code=%d", err, code)
	}
	if string(out) != "ping" {
		t.Fatalf("out=%q", out)
	}
}

func TestExecRunnerPropagatesExitCode(t *testing.T) {
	r := newExecRunner("sh")
	_, _, code, err := r.Run(context.Background(), []string{"-c", "exit 7"}, nil)
	if err == nil {
		t.Fatal("want error for non-zero exit")
	}
	if code != 7 {
		t.Fatalf("code=%d want 7", code)
	}
}

func TestRunError_RedactsArgs(t *testing.T) {
	e := &runError{bin: "container", args: []string{"create", "-e", "DB_PASSWORD=hunter2"}, code: 1, stderr: "boom"}
	msg := e.Error()
	assert.Equal(t, "container create: exit 1: boom", msg)
	assert.NotContains(t, msg, "hunter2")
}

func TestCliProper(t *testing.T) {
	e := &runError{bin: "bin", args: []string{"a"}, code: 1, stderr: "err", cause: os.ErrClosed}
	assert.ErrorIs(t, e.Unwrap(), os.ErrClosed)
	assert.Equal(t, "bin a: exit 1: err", e.Error())

	runner := newExecRunner("echo")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	cmd, _, err := runner.Start(ctx, []string{"hello"}, nil)
	require.NoError(t, err)
	assert.NotNil(t, cmd)
}

func TestCLIStartAndRun(t *testing.T) {
	r := newExecRunner("echo")

	// Test Run
	stdout, _, code, err := r.Run(context.Background(), []string{"hello"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Contains(t, string(stdout), "hello")

	// Test Start
	cmd, pr, err := r.Start(context.Background(), []string{"hello"}, nil)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.NotNil(t, pr)
	out, _ := io.ReadAll(pr)
	assert.Contains(t, string(out), "hello")
}
