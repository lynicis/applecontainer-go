package wait

import (
	"context"
	"time"
)

// FileStrategy waits for a file to exist inside the container.
type FileStrategy struct {
	Path         string
	PollInterval time.Duration
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
	checkReady := func() bool {
		rc, err := target.CopyFileFromContainer(ctx, s.Path)
		if err != nil {
			return false
		}
		_ = rc.Close()
		return true
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
