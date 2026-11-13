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
		status, code, err := target.State(ctx)
		return err == nil && status == "running" && code == 0
	}

	return pollReady(ctx, s.PollInterval, checkReady)
}
