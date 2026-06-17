package applecontainer

import (
	"context"
	"testing"
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
