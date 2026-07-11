package wait

import (
	"context"
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
			// Container might not be created or might be gone already.
			return true
		}
		return status != "running"
	}

	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if checkReady() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
