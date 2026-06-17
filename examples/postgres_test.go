//go:build integration
// +build integration

package examples

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
)

func TestPostgresIntegration(t *testing.T) {
	applecontainer.SkipIfProviderNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dburl := func(host string, port int) string {
		return fmt.Sprintf("postgres://postgres:postgres@%s:%d/postgres?sslmode=disable", host, port)
	}

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
		t.Fatalf("failed to start postgres: %v", err)
	}
	applecontainer.CleanupContainer(t, c)

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}
	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	db, err := sql.Open("pgx", dburl(host, port))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var val int
	err = db.QueryRowContext(ctx, "SELECT 1").Scan(&val)
	if err != nil {
		t.Fatalf("failed to query db: %v", err)
	}

	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
}
