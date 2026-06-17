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
		if err := runStrategy(ctx, strat, target); err != nil {
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

// runStrategy executes a wait strategy, wrapping it in a context timeout
// if the strategy implements StrategyTimeout with a custom timeout.
func runStrategy(ctx context.Context, strat Strategy, target StrategyTarget) error {
	timeout := 60 * time.Second
	if st, ok := strat.(StrategyTimeout); ok && st.Timeout() > 0 {
		timeout = st.Timeout()
	}

	subCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return strat.WaitUntilReady(subCtx, target)
}
