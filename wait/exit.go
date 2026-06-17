package wait

import (
	"context"
	"time"
)

// ExitStrategy waits for a container to stop/exit (i.e. status is not "running").
type ExitStrategy struct {
	startupTimeout time.Duration
	PollInterval   time.Duration
}

// Timeout returns the custom timeout for this strategy.
func (s *ExitStrategy) Timeout() time.Duration {
	return s.startupTimeout
}

// WithStartupTimeout sets the custom startup timeout.
func (s *ExitStrategy) WithStartupTimeout(d time.Duration) *ExitStrategy {
	s.startupTimeout = d
	return s
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
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := target.StateStatus(ctx)
			if err != nil {
				// Container might not be created or might be gone already
				return nil
			}

			if status != "running" {
				return nil
			}
		}
	}
}
