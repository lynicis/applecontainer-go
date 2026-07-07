package wait

import (
	"bytes"
	"context"
	"io"
	"time"
)

// ExecStrategy waits for a command inside the container to succeed.
type ExecStrategy struct {
	Cmd             []string
	ExitCodeMatcher func(int) bool
	ResponseMatcher func(io.Reader) bool
	PollInterval    time.Duration
}

// WithExitCodeMatcher sets a custom matcher for the command's exit code.
func (s *ExecStrategy) WithExitCodeMatcher(matcher func(int) bool) *ExecStrategy {
	s.ExitCodeMatcher = matcher
	return s
}

// WithResponseMatcher sets a custom matcher for the command's output.
func (s *ExecStrategy) WithResponseMatcher(matcher func(io.Reader) bool) *ExecStrategy {
	s.ResponseMatcher = matcher
	return s
}

// WithPollInterval sets the polling interval.
func (s *ExecStrategy) WithPollInterval(d time.Duration) *ExecStrategy {
	s.PollInterval = d
	return s
}

// ForExec creates an ExecWaitStrategy for the command.
func ForExec(cmd []string) *ExecStrategy {
	return &ExecStrategy{
		Cmd:             cmd,
		ExitCodeMatcher: func(code int) bool { return code == 0 },
		PollInterval:    100 * time.Millisecond,
	}
}

// WaitUntilReady executes the command repeatedly until the exit code and output match, or times out.
func (s *ExecStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			code, out, err := target.Exec(ctx, s.Cmd)
			if err != nil {
				continue
			}

			ok := s.ExitCodeMatcher(code)
			if ok && s.ResponseMatcher != nil {
				ok = s.ResponseMatcher(bytes.NewReader(out))
			}

			if ok {
				return nil
			}
		}
	}
}
