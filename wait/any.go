package wait

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ForAnyStrategy executes a list of wait strategies concurrently, succeeding when any one succeeds.
type ForAnyStrategy struct {
	Strategies []Strategy
	Deadline   time.Duration
}

// WaitUntilReady runs all sub-strategies concurrently.
func (s *ForAnyStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	if s.Deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Deadline)
		defer cancel()
	}

	if len(s.Strategies) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errs := make([]error, len(s.Strategies))
	successChan := make(chan struct{}, 1)

	for i, strat := range s.Strategies {
		wg.Add(1)
		go func(idx int, st Strategy) {
			defer wg.Done()
			err := st.WaitUntilReady(ctx, target)
			if err == nil {
				select {
				case successChan <- struct{}{}:
					cancel() // cancel all other strategies
				default:
				}
			} else {
				errs[idx] = err
			}
		}(i, strat)
	}

	// Wait for all to complete in a separate goroutine.
	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	select {
	case <-successChan:
		return nil
	case <-doneChan:
		// Check if at least one succeeded (should not happen if doneChan completes without successChan,
		// but is a safe fallback).
		// Return combined errors.
		combinedErr := errors.Join(errs...)
		if combinedErr == nil {
			return errors.New("any wait strategy failed: no strategies ran")
		}
		return combinedErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WithDeadline sets a deadline for the wait strategy.
func (s *ForAnyStrategy) WithDeadline(d time.Duration) *ForAnyStrategy {
	s.Deadline = d
	return s
}

// ForAny creates a composite wait strategy where any success triggers readiness.
func ForAny(strategies ...Strategy) *ForAnyStrategy {
	return &ForAnyStrategy{Strategies: strategies}
}
