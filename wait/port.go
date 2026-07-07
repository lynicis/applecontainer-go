package wait

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// PortStrategy waits for a port to start listening.
type PortStrategy struct {
	Port         string
	PollInterval time.Duration
}

// WithPollInterval sets the polling interval.
func (s *PortStrategy) WithPollInterval(d time.Duration) *PortStrategy {
	s.PollInterval = d
	return s
}

// ForListeningPort waits for the given port to be open.
func ForListeningPort(port string) *PortStrategy {
	return &PortStrategy{
		Port:         port,
		PollInterval: 100 * time.Millisecond,
	}
}

// WaitUntilReady dials the container port repeatedly until success or context timeout.
func (s *PortStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	pNum, proto := ParsePort(s.Port)
	if pNum <= 0 {
		return fmt.Errorf("invalid port specification: %s", s.Port)
	}

	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			host, err := target.Host(ctx)
			if err != nil {
				continue
			}

			mappedPort, err := target.MappedPort(ctx, s.Port)
			if err != nil {
				continue
			}

			address := net.JoinHostPort(host, strconv.Itoa(mappedPort))
			dialer := net.Dialer{}
			conn, err := dialer.DialContext(ctx, proto, address)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}

// ParsePort parses a port number and protocol from a port string (e.g. "80/tcp" or "80").
func ParsePort(port string) (int, string) {
	parts := strings.Split(port, "/")
	pNum, _ := strconv.Atoi(parts[0])
	proto := "tcp"
	if len(parts) > 1 && parts[1] != "" {
		proto = strings.ToLower(parts[1])
	}
	return pNum, proto
}
