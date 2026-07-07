package wait

import (
	"context"
	"io"
)

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
	StateStatus(ctx context.Context) (string, error)
	StateExitCode(ctx context.Context) (int, error)
	CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error)
}
