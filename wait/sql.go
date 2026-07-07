package wait

import (
	"context"
	"database/sql"
	"time"
)

// SQLStrategy waits for a database to accept queries.
type SQLStrategy struct {
	Port         string
	Driver       string
	DBURL        func(host string, port int) string
	Query        string
	PollInterval time.Duration
}

// WithQuery sets the validation query to run.
func (s *SQLStrategy) WithQuery(q string) *SQLStrategy {
	s.Query = q
	return s
}

// WithPollInterval sets the polling interval.
func (s *SQLStrategy) WithPollInterval(d time.Duration) *SQLStrategy {
	s.PollInterval = d
	return s
}

// ForSQL creates a SQLStrategy.
func ForSQL(port string, driver string, dburl func(host string, port int) string) *SQLStrategy {
	return &SQLStrategy{
		Port:         port,
		Driver:       driver,
		DBURL:        dburl,
		Query:        "SELECT 1",
		PollInterval: 100 * time.Millisecond,
	}
}

// WaitUntilReady opens a DB connection and pings/queries it repeatedly until success, or times out.
func (s *SQLStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
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

			urlStr := s.DBURL(host, mappedPort)
			db, err := sql.Open(s.Driver, urlStr)
			if err != nil {
				continue
			}

			// We need to execute the query and ensure it works
			err = func() error {
				defer func() { _ = db.Close() }()
				connCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()

				if err := db.PingContext(connCtx); err != nil {
					return err
				}

				rows, err := db.QueryContext(connCtx, s.Query)
				if err != nil {
					return err
				}
				_ = rows.Close()
				return nil
			}()

			if err == nil {
				return nil
			}
		}
	}
}
