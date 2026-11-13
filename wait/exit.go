package wait

import (
	"context"
	"strings"
	"time"
)

// ExitStrategy waits for a container to stop/exit (i.e. status is not "running").
type ExitStrategy struct {
	PollInterval time.Duration
}

// WithPollInterval sets the polling interval.
func (s *ExitStrategy) WithPollInterval(d time.Duration) *ExitStrategy {
	s.PollInterval = d
	return s
}

// ForExit creates an ExitStrategy.
func ForExit() *ExitStrategy {
	return &ExitStrategy{
		PollInterval: 100 * time.Millisecond,
	}
}

// WaitUntilReady blocks until the container's status is not "running".
func (s *ExitStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	checkReady := func() bool {
		status, err := target.StateStatus(ctx)
		if err != nil {
			// A gone container has exited; other inspect errors are transient.
			return isGone(err)
		}
		return status != "running"
	}

	return pollReady(ctx, s.PollInterval, checkReady)
}

// isGone intentionally duplicates root's isNotFoundError because wait cannot import
// the root package, and it is substring-based because the CLI exposes no structured
// not-found error.
func isGone(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such") ||
		strings.Contains(msg, "does not exist")
}
