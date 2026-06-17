package wait

import (
	"context"
	"io"
	"time"
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
}

// StrategyTimeout defines an interface for strategies with custom timeouts.
type StrategyTimeout interface {
	Timeout() time.Duration
}

// ForAllStrategy executes a list of wait strategies sequentially.
type ForAllStrategy struct {
	Strategies []Strategy
	Deadline   time.Duration
}

// WaitUntilReady runs all sub-strategies.
func (s *ForAllStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	for _, strat := range s.Strategies {
		if err := strat.WaitUntilReady(ctx, target); err != nil {
			return err
		}
	}
	return nil
}

// WithDeadline sets a deadline for the wait strategy.
func (s *ForAllStrategy) WithDeadline(d time.Duration) *ForAllStrategy {
	s.Deadline = d
	return s
}

// ForAll creates a composite wait strategy.
func ForAll(strategies ...Strategy) *ForAllStrategy {
	return &ForAllStrategy{Strategies: strategies}
}
