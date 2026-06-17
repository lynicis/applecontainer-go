package wait

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// LogStrategy waits for a specific log pattern to appear in the container's logs.
type LogStrategy struct {
	Pattern        string
	Occurrence     int
	IsRegexp       bool
	startupTimeout time.Duration
	PollInterval   time.Duration
}

// Timeout returns the custom timeout for this strategy.
func (s *LogStrategy) Timeout() time.Duration {
	return s.startupTimeout
}

// AsRegexp configures the pattern to be treated as a regular expression.
func (s *LogStrategy) AsRegexp() *LogStrategy {
	s.IsRegexp = true
	return s
}

// WithOccurrence sets the number of times the pattern must appear.
func (s *LogStrategy) WithOccurrence(n int) *LogStrategy {
	s.Occurrence = n
	return s
}

// WithStartupTimeout sets the custom startup timeout.
func (s *LogStrategy) WithStartupTimeout(d time.Duration) *LogStrategy {
	s.startupTimeout = d
	return s
}

// WithPollInterval sets the polling interval.
func (s *LogStrategy) WithPollInterval(d time.Duration) *LogStrategy {
	s.PollInterval = d
	return s
}

// ForLog creates a wait strategy that monitors container logs.
func ForLog(pattern string) *LogStrategy {
	return &LogStrategy{
		Pattern:    pattern,
		Occurrence: 1,
	}
}

// WaitUntilReady reads the target logs and blocks until the occurrence constraint is met or context expires.
func (s *LogStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	reader, err := target.Logs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get target logs: %w", err)
	}
	defer reader.Close()

	var re *regexp.Regexp
	if s.IsRegexp {
		re, err = regexp.Compile(s.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regexp pattern %q: %w", s.Pattern, err)
		}
	}

	count := 0
	lineChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			lineChan <- scanner.Text()
		}
		close(lineChan)
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, ok := <-lineChan:
			if !ok {
				select {
				case err := <-errChan:
					return fmt.Errorf("log stream ended with error before pattern %q occurred %d times (found %d): %w", s.Pattern, s.Occurrence, count, err)
				default:
					return fmt.Errorf("log stream ended before pattern %q occurred %d times (found %d): %w", s.Pattern, s.Occurrence, count, io.EOF)
				}
			}
			matched := false
			if s.IsRegexp {
				matched = re.MatchString(line)
			} else {
				matched = strings.Contains(line, s.Pattern)
			}

			if matched {
				count++
				if count >= s.Occurrence {
					return nil
				}
			}
		}
	}
}
