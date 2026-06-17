package wait

import (
	"context"
	"time"
)

// FileStrategy waits for a file to exist inside the container.
type FileStrategy struct {
	Path           string
	startupTimeout time.Duration
	PollInterval   time.Duration
}

// Timeout returns the custom timeout for this strategy.
func (s *FileStrategy) Timeout() time.Duration {
	return s.startupTimeout
}

// WithStartupTimeout sets the custom startup timeout.
func (s *FileStrategy) WithStartupTimeout(d time.Duration) *FileStrategy {
	s.startupTimeout = d
	return s
}

// WithPollInterval sets the polling interval.
func (s *FileStrategy) WithPollInterval(d time.Duration) *FileStrategy {
	s.PollInterval = d
	return s
}

// ForFile creates a FileStrategy.
func ForFile(path string) *FileStrategy {
	return &FileStrategy{
		Path:         path,
		PollInterval: 100 * time.Millisecond,
	}
}

// WaitUntilReady checks for file presence in the container using CopyFileFromContainer.
func (s *FileStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			rc, err := target.CopyFileFromContainer(ctx, s.Path)
			if err == nil {
				_ = rc.Close()
				return nil
			}
		}
	}
}
