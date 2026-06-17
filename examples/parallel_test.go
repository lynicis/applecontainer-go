//go:build integration
// +build integration

package examples

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
)

func TestParallelPostgresIntegration(t *testing.T) {
	applecontainer.SkipIfProviderNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	dburl := func(host string, port int) string {
		return fmt.Sprintf("postgres://postgres:postgres@%s:%d/postgres?sslmode=disable", host, port)
	}

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c, err := applecontainer.Run(ctx, "postgres:alpine",
				applecontainer.WithExposedPorts("5432"),
				applecontainer.WithEnv(map[string]string{
					"POSTGRES_USER":     "postgres",
					"POSTGRES_PASSWORD": "postgres",
					"POSTGRES_DB":       "postgres",
				}),
				applecontainer.WithWaitStrategy(wait.ForSQL("5432", "pgx", dburl)),
			)
			if err != nil {
				t.Errorf("instance %d failed to start: %v", idx, err)
				return
			}
			applecontainer.CleanupContainer(t, c)

			host, err := c.Host(ctx)
			if err != nil {
				t.Errorf("instance %d failed to get host: %v", idx, err)
				return
			}

			db, err := sql.Open("pgx", dburl(host, 5432))
			if err != nil {
				t.Errorf("instance %d failed to open db: %v", idx, err)
				return
			}
			defer db.Close()

			var val int
			err = db.QueryRowContext(ctx, "SELECT 1").Scan(&val)
			if err != nil {
				t.Errorf("instance %d failed to query: %v", idx, err)
				return
			}
			if val != 1 {
				t.Errorf("instance %d expected 1, got %d", idx, val)
			}
		}(i)
	}
	wg.Wait()
}
