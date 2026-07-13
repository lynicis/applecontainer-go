package benchmarks

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	wait2 "github.com/testcontainers/testcontainers-go/wait"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type BenchModel struct {
	ID  int `gorm:"primaryKey"`
	Val string
}

func (BenchModel) TableName() string {
	return "bench_models"
}

func setupPostgres(t testing.TB, rt Runtime) (string, func()) {
	t.Helper()

	switch rt {
	case AppleContainer:
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		c, err := applecontainer.Run(ctx, "postgres:15-alpine",
			applecontainer.WithExposedPorts("5432"),
			applecontainer.WithEnv(map[string]string{
				"POSTGRES_DB":       "bench",
				"POSTGRES_USER":     "bench",
				"POSTGRES_PASSWORD": "bench",
			}),
			applecontainer.WithWaitingFor(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2),
			),
		)
		if err != nil {
			t.Fatalf("applecontainer postgres: %v", err)
		}
		endpoint, err := c.Endpoint(ctx, "5432")
		if err != nil {
			t.Fatalf("applecontainer endpoint: %v", err)
		}
		connStr := fmt.Sprintf("postgres://bench:bench@%s/bench?sslmode=disable", endpoint)

		// Create table
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			t.Fatalf("sql open: %v", err)
		}
		defer db.Close()
		if _, err := db.Exec("CREATE TABLE bench_models (id SERIAL PRIMARY KEY, val TEXT)"); err != nil {
			t.Fatalf("create table: %v", err)
		}
		return connStr, func() { _ = c.Terminate(context.Background()) }

	case TestcontainersGo:
		pgContainer, err := postgres.Run(context.Background(), "postgres:15-alpine",
			postgres.WithDatabase("bench"),
			postgres.WithUsername("bench"),
			postgres.WithPassword("bench"),
			testcontainers.WithWaitStrategy(
				wait2.ForLog("database system is ready to accept connections").
					WithOccurrence(2).WithStartupTimeout(30*time.Second),
			),
		)
		if err != nil {
			t.Fatalf("tc postgres: %v", err)
		}
		connStr, err := pgContainer.ConnectionString(context.Background(), "sslmode=disable")
		if err != nil {
			t.Fatalf("conn string: %v", err)
		}

		// Create table
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			t.Fatalf("sql open: %v", err)
		}
		defer db.Close()
		if _, err := db.Exec("CREATE TABLE bench_models (id SERIAL PRIMARY KEY, val TEXT)"); err != nil {
			t.Fatalf("create table: %v", err)
		}
		return connStr, func() { _ = pgContainer.Terminate(context.Background()) }
	}
	panic("unreachable")
}

// runDriverBenchmarks exercises pgx, database/sql (lib/pq), and GORM against a Postgres connStr.
// Pre-condition: the bench_models table already exists.
func runDriverBenchmarks(b *testing.B, connStr string) {
	ctx := context.Background()

	// 1. pgx setup
	pgxPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		b.Fatalf("pgxpool.New failed: %v", err)
	}
	defer pgxPool.Close()

	// 2. pq setup
	pqDB, err := sql.Open("postgres", connStr)
	if err != nil {
		b.Fatalf("sql.Open pq failed: %v", err)
	}
	defer func() { _ = pqDB.Close() }()

	// 3. gorm setup
	gormDB, err := gorm.Open(gormpg.Open(connStr), &gorm.Config{})
	if err != nil {
		b.Fatalf("gorm.Open failed: %v", err)
	}

	// Pre-insert some data for selects
	for i := 0; i < 1000; i++ {
		_, err = pqDB.Exec("INSERT INTO bench_models (val) VALUES ($1)", fmt.Sprintf("val-%d", i))
		if err != nil {
			b.Fatalf("pre-insert failed: %v", err)
		}
	}

	b.Run("pgx-insert", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pgxPool.Exec(ctx, "INSERT INTO bench_models (val) VALUES ($1)", "pgx-ins")
		}
	})

	b.Run("pq-insert", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pqDB.Exec("INSERT INTO bench_models (val) VALUES ($1)", "pq-ins")
		}
	})

	b.Run("gorm-insert", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = gormDB.Create(&BenchModel{Val: "gorm-ins"}).Error
		}
	})

	// Bulk insert (100 rows)
	b.Run("pgx-bulk-insert-100", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := "INSERT INTO bench_models (val) VALUES "
			args := make([]any, 100)
			for j := 0; j < 100; j++ {
				if j > 0 {
					query += ","
				}
				query += fmt.Sprintf("($%d)", j+1)
				args[j] = "pgx-bulk"
			}
			_, _ = pgxPool.Exec(ctx, query, args...)
		}
	})

	b.Run("pq-bulk-insert-100", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := "INSERT INTO bench_models (val) VALUES "
			args := make([]any, 100)
			for j := 0; j < 100; j++ {
				if j > 0 {
					query += ","
				}
				query += fmt.Sprintf("($%d)", j+1)
				args[j] = "pq-bulk"
			}
			_, _ = pqDB.Exec(query, args...)
		}
	})

	b.Run("gorm-bulk-insert-100", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rows := make([]BenchModel, 100)
			for j := range rows {
				rows[j].Val = "gorm-bulk"
			}
			_ = gormDB.CreateInBatches(&rows, 100).Error
		}
	})

	// Select by PK
	b.Run("pgx-select-pk", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var val string
			_ = pgxPool.QueryRow(ctx, "SELECT val FROM bench_models WHERE id = $1", 1).Scan(&val)
		}
	})

	b.Run("pq-select-pk", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var val string
			_ = pqDB.QueryRow("SELECT val FROM bench_models WHERE id = $1", 1).Scan(&val)
		}
	})

	b.Run("gorm-select-pk", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var m BenchModel
			_ = gormDB.First(&m, 1).Error
		}
	})

	// Select with filter (LIKE query, returns multiple rows)
	b.Run("pgx-select-filter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rows, _ := pgxPool.Query(ctx, "SELECT id, val FROM bench_models WHERE val LIKE $1 LIMIT 50", "val-%")
			if rows != nil {
				rows.Close()
			}
		}
	})

	b.Run("pq-select-filter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rows, _ := pqDB.Query("SELECT id, val FROM bench_models WHERE val LIKE $1 LIMIT 50", "val-%")
			if rows != nil {
				_ = rows.Close()
			}
		}
	})

	b.Run("gorm-select-filter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var ms []BenchModel
			_ = gormDB.Where("val LIKE ?", "val-%").Limit(50).Find(&ms).Error
		}
	})

	// Update
	b.Run("pgx-update", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pgxPool.Exec(ctx, "UPDATE bench_models SET val = $1 WHERE id = $2", "pgx-upd", 1)
		}
	})

	b.Run("pq-update", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pqDB.Exec("UPDATE bench_models SET val = $1 WHERE id = $2", "pq-upd", 1)
		}
	})

	b.Run("gorm-update", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = gormDB.Model(&BenchModel{ID: 1}).Update("val", "gorm-upd").Error
		}
	})

	// Delete
	b.Run("pgx-delete", func(b *testing.B) {
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pgxPool.Exec(ctx, "INSERT INTO bench_models (val) VALUES ($1)", "del")
		}
		b.StartTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = pgxPool.Exec(ctx, "DELETE FROM bench_models WHERE id = (SELECT id FROM bench_models LIMIT 1)")
		}
	})

	b.Run("pq-delete", func(b *testing.B) {
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pqDB.Exec("INSERT INTO bench_models (val) VALUES ($1)", "del")
		}
		b.StartTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = pqDB.Exec("DELETE FROM bench_models WHERE id = (SELECT id FROM bench_models LIMIT 1)")
		}
	})

	b.Run("gorm-delete", func(b *testing.B) {
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			_ = gormDB.Create(&BenchModel{Val: "del"}).Error
		}
		b.StartTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = gormDB.Exec("DELETE FROM bench_models WHERE id = (SELECT id FROM bench_models LIMIT 1)")
		}
	})
}

func BenchmarkDrivers(b *testing.B) {
	if b.N < 1 {
		return
	}

	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		prePull(b, rt, "postgres:15-alpine")
		connStr, teardown := setupPostgres(b, rt)
		defer teardown()
		runDriverBenchmarks(b, connStr)
	})
}
