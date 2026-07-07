package wait

import (
	"context"
	"time"
)

// ForAllStrategy executes a list of wait strategies sequentially.
type ForAllStrategy struct {
	Strategies []Strategy
	Deadline   time.Duration
}

// WaitUntilReady runs all sub-strategies sequentially.
func (s *ForAllStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	if s.Deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Deadline)
		defer cancel()
	}

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
