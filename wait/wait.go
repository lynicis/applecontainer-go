package wait

import (
	"context"
	"io"
	"time"
)

const defaultPollInterval = 100 * time.Millisecond

// pollReady runs check immediately, then every interval until it returns true or ctx is done.
// A non-positive interval falls back to defaultPollInterval (100ms).
func pollReady(ctx context.Context, interval time.Duration, check func() bool) error {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if check() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Strategy defines the interface for container wait strategies.
type Strategy interface {
	WaitUntilReady(ctx context.Context, target StrategyTarget) error
}

// StrategyTarget defines the interface that a container must satisfy to use wait strategies.
type StrategyTarget interface {
	Host(ctx context.Context) (string, error)
	MappedPort(ctx context.Context, port string) (int, error)
	Logs(ctx context.Context) (io.ReadCloser, error)
	Exec(ctx context.Context, cmd []string, opts ...any) (int, []byte, error)
	State(ctx context.Context) (status string, exitCode int, err error)
	StateStatus(ctx context.Context) (string, error)
	StateExitCode(ctx context.Context) (int, error)
	CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error)
}
