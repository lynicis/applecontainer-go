package wait

import (
	"context"
	"time"
)

// HealthStrategy waits for the container status to be "running" and exit code to be 0.
type HealthStrategy struct {
	PollInterval time.Duration
}

// WithPollInterval sets the polling interval.
func (s *HealthStrategy) WithPollInterval(d time.Duration) *HealthStrategy {
	s.PollInterval = d
	return s
}

// ForHealth creates a HealthStrategy.
func ForHealth() *HealthStrategy {
	return &HealthStrategy{
		PollInterval: 100 * time.Millisecond,
	}
}

// WaitUntilReady blocks until the container is running and has exit code 0.
func (s *HealthStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	checkReady := func() bool {
		status, err := target.StateStatus(ctx)
		if err != nil {
			return false
		}

		code, err := target.StateExitCode(ctx)
		if err != nil {
			return false
		}
		return status == "running" && code == 0
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
