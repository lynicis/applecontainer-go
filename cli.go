package applecontainer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// commandRunner is the single seam to the `container` binary. Tests fake it.
type commandRunner interface {
	Run(ctx context.Context, args []string, stdin []byte) (stdout, stderr []byte, exitCode int, err error)
	Start(ctx context.Context, args []string, stdin io.Reader) (cmd *exec.Cmd, stdout, stderr io.Reader, err error)
}

// execRunner implements commandRunner via os/exec against a fixed binary path.
type execRunner struct{ bin string }

func newExecRunner(bin string) *execRunner { return &execRunner{bin: bin} }

func (r *execRunner) Run(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, r.bin, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		code = -1
	}
	if code != 0 {
		err = &runError{bin: r.bin, args: args, code: code, stderr: errb.String(), cause: err}
	}
	return out.Bytes(), errb.Bytes(), code, err
}

func (r *execRunner) Start(ctx context.Context, args []string, stdin io.Reader) (*exec.Cmd, io.Reader, io.Reader, error) {
	cmd := exec.CommandContext(ctx, r.bin, args...)
	cmd.Stdin = stdin
	outPR, outPW := io.Pipe()
	errPR, errPW := io.Pipe()
	cmd.Stdout, cmd.Stderr = outPW, errPW
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	go func() {
		_ = cmd.Wait()
		_ = outPW.Close()
		_ = errPW.Close()
	}()
	return cmd, outPR, errPR, nil
}

type runError struct {
	bin    string
	args   []string
	code   int
	stderr string
	cause  error
}

func (e *runError) Error() string {
	return fmt.Sprintf("%s %s: exit %d: %s", e.bin, strings.Join(e.args, " "), e.code, e.stderr)
}

func (e *runError) Unwrap() error { return e.cause }
