package wait

import (
	"context"
	"time"
)

// HealthStrategy waits for the container status to be "running" and exit code to be 0.
type HealthStrategy struct {
	startupTimeout time.Duration
	PollInterval   time.Duration
}

// Timeout returns the custom timeout for this strategy.
func (s *HealthStrategy) Timeout() time.Duration {
	return s.startupTimeout
}

// WithStartupTimeout sets the custom startup timeout.
func (s *HealthStrategy) WithStartupTimeout(d time.Duration) *HealthStrategy {
	s.startupTimeout = d
	return s
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
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := target.StateStatus(ctx)
			if err != nil {
				continue
			}

			code, err := target.StateExitCode(ctx)
			if err != nil {
				continue
			}

			if status == "running" && code == 0 {
				return nil
			}
		}
	}
}
