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
	checkReady := func() bool {
		host, err := target.Host(ctx)
		if err != nil {
			return false
		}

		mappedPort, err := target.MappedPort(ctx, s.Port)
		if err != nil {
			return false
		}

		db, err := sql.Open(s.Driver, s.DBURL(host, mappedPort))
		if err != nil {
			return false
		}
		defer func() { _ = db.Close() }()

		connCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := db.PingContext(connCtx); err != nil {
			return false
		}

		rows, err := db.QueryContext(connCtx, s.Query)
		if err != nil {
			return false
		}
		_ = rows.Close()
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
