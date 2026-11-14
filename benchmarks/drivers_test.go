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

// createTable creates the fixture table. The val index keeps the per-sample
// insert cleanup (DELETE ... WHERE val = $1) cheap; both runtimes share it.
func createTable(t testing.TB, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("CREATE TABLE bench_models (id SERIAL PRIMARY KEY, val TEXT)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("CREATE INDEX bench_models_val_idx ON bench_models (val)"); err != nil {
		t.Fatalf("create index: %v", err)
	}
}

func setupPostgres(t testing.TB, rt Runtime) (string, func()) {
	t.Helper()

	switch rt {
	case AppleContainer:
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		c, err := applecontainer.Run(ctx, imagePostgres,
			applecontainer.WithExposedPorts("5432"),
			applecontainer.WithEnv(map[string]string{
				"POSTGRES_DB":       "bench",
				"POSTGRES_USER":     "bench",
				"POSTGRES_PASSWORD": "bench",
			}),
			applecontainer.WithWaitingFor(wait.ForAll(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			).WithDeadline(120*time.Second)),
		)
		if err != nil {
			t.Fatalf("applecontainer postgres: %v", err)
		}
		endpoint, err := c.Endpoint(ctx, "5432")
		if err != nil {
			t.Fatalf("applecontainer endpoint: %v", err)
		}
		connStr := fmt.Sprintf("postgres://bench:bench@%s/bench?sslmode=disable", endpoint)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			t.Fatalf("sql open: %v", err)
		}
		defer db.Close()
		createTable(t, db)
		return connStr, func() { _ = c.Terminate(context.Background()) }

	case TestcontainersGo:
		pgContainer, err := postgres.Run(context.Background(), imagePostgres,
			postgres.WithDatabase("bench"),
			postgres.WithUsername("bench"),
			postgres.WithPassword("bench"),
			testcontainers.WithWaitStrategy(
				wait2.ForLog("database system is ready to accept connections").
					WithOccurrence(2).WithStartupTimeout(120*time.Second),
			),
		)
		if err != nil {
			t.Fatalf("tc postgres: %v", err)
		}
		connStr, err := pgContainer.ConnectionString(context.Background(), "sslmode=disable")
		if err != nil {
			t.Fatalf("conn string: %v", err)
		}

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			t.Fatalf("sql open: %v", err)
		}
		defer db.Close()
		createTable(t, db)
		return connStr, func() { _ = pgContainer.Terminate(context.Background()) }
	}
	panic("unreachable")
}

const bulkRows = 100

// bulkInsertQuery builds "INSERT ... VALUES ($1),($2),..." once so the timed
// loop measures the driver, not string building.
func bulkInsertQuery(n int) string {
	q := "INSERT INTO bench_models (val) VALUES "
	for j := range n {
		if j > 0 {
			q += ","
		}
		q += fmt.Sprintf("($%d)", j+1)
	}
	return q
}

// cleanupRows removes the rows a timed insert benchmark wrote. It runs outside
// the timer so insert samples do not grow the fixture table and skew later
// select/delete benchmarks.
func cleanupRows(b *testing.B, db *sql.DB, val string) {
	b.StopTimer()
	if _, err := db.Exec("DELETE FROM bench_models WHERE val = $1", val); err != nil {
		b.Fatalf("cleanup %s: %v", val, err)
	}
}

// runDriverBenchmarks exercises pgx, database/sql (lib/pq), and GORM against a Postgres connStr.
// Pre-condition: the bench_models table already exists and contains the seed rows.
func runDriverBenchmarks(b *testing.B, connStr string) {
	ctx := context.Background()

	pgxPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		b.Fatalf("pgxpool.New failed: %v", err)
	}
	defer pgxPool.Close()

	pqDB, err := sql.Open("postgres", connStr)
	if err != nil {
		b.Fatalf("sql.Open pq failed: %v", err)
	}
	defer func() { _ = pqDB.Close() }()

	gormDB, err := gorm.Open(gormpg.Open(connStr), &gorm.Config{})
	if err != nil {
		b.Fatalf("gorm.Open failed: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		b.Fatalf("gorm db: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if _, err := pqDB.Exec("INSERT INTO bench_models (val) SELECT 'val-' || g FROM generate_series(1, 1000) AS g"); err != nil {
		b.Fatalf("seed: %v", err)
	}

	bulkQuery := bulkInsertQuery(bulkRows)
	pgxBulkArgs := make([]any, bulkRows)
	pqBulkArgs := make([]any, bulkRows)
	for j := range bulkRows {
		pgxBulkArgs[j] = "pgx-bulk"
		pqBulkArgs[j] = "pq-bulk"
	}

	b.Run("pgx-insert", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := pgxPool.Exec(ctx, "INSERT INTO bench_models (val) VALUES ($1)", "pgx-ins"); err != nil {
				b.Fatal(err)
			}
		}
		cleanupRows(b, pqDB, "pgx-ins")
	})

	b.Run("pq-insert", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := pqDB.Exec("INSERT INTO bench_models (val) VALUES ($1)", "pq-ins"); err != nil {
				b.Fatal(err)
			}
		}
		cleanupRows(b, pqDB, "pq-ins")
	})

	b.Run("gorm-insert", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if err := gormDB.Create(&BenchModel{Val: "gorm-ins"}).Error; err != nil {
				b.Fatal(err)
			}
		}
		cleanupRows(b, pqDB, "gorm-ins")
	})

	b.Run("pgx-bulk-insert-100", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := pgxPool.Exec(ctx, bulkQuery, pgxBulkArgs...); err != nil {
				b.Fatal(err)
			}
		}
		cleanupRows(b, pqDB, "pgx-bulk")
	})

	b.Run("pq-bulk-insert-100", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := pqDB.Exec(bulkQuery, pqBulkArgs...); err != nil {
				b.Fatal(err)
			}
		}
		cleanupRows(b, pqDB, "pq-bulk")
	})

	b.Run("gorm-bulk-insert-100", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			rows := make([]BenchModel, bulkRows)
			for j := range rows {
				rows[j].Val = "gorm-bulk"
			}
			if err := gormDB.CreateInBatches(&rows, bulkRows).Error; err != nil {
				b.Fatal(err)
			}
		}
		cleanupRows(b, pqDB, "gorm-bulk")
	})

	b.Run("pgx-select-pk", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			var val string
			if err := pgxPool.QueryRow(ctx, "SELECT val FROM bench_models WHERE id = $1", 1).Scan(&val); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pq-select-pk", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			var val string
			if err := pqDB.QueryRow("SELECT val FROM bench_models WHERE id = $1", 1).Scan(&val); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm-select-pk", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			var m BenchModel
			if err := gormDB.First(&m, 1).Error; err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pgx-select-filter", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			rows, err := pgxPool.Query(ctx, "SELECT id, val FROM bench_models WHERE val LIKE $1 LIMIT 50", "val-%")
			if err != nil {
				b.Fatal(err)
			}
			rows.Close()
		}
	})

	b.Run("pq-select-filter", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			rows, err := pqDB.Query("SELECT id, val FROM bench_models WHERE val LIKE $1 LIMIT 50", "val-%")
			if err != nil {
				b.Fatal(err)
			}
			if err := rows.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm-select-filter", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			var ms []BenchModel
			if err := gormDB.Where("val LIKE ?", "val-%").Limit(50).Find(&ms).Error; err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pgx-update", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := pgxPool.Exec(ctx, "UPDATE bench_models SET val = $1 WHERE id = $2", "pgx-upd", 1); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pq-update", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := pqDB.Exec("UPDATE bench_models SET val = $1 WHERE id = $2", "pq-upd", 1); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm-update", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if err := gormDB.Model(&BenchModel{ID: 1}).Update("val", "gorm-upd").Error; err != nil {
				b.Fatal(err)
			}
		}
	})

	// Deletes insert their own rows first so they never consume the id = 1 row
	// used by the select and update benchmarks.
	b.Run("pgx-delete", func(b *testing.B) {
		b.StopTimer()
		rows, err := pgxPool.Query(ctx, "INSERT INTO bench_models (val) SELECT 'del' FROM generate_series(1, $1::int) RETURNING id", b.N)
		if err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		ids := make([]int64, 0, b.N)
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				b.Fatalf("scan id: %v", err)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		rows.Close()
		b.ReportAllocs()
		b.ResetTimer()
		b.StartTimer()
		for _, id := range ids {
			if _, err := pgxPool.Exec(ctx, "DELETE FROM bench_models WHERE id = $1", id); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pq-delete", func(b *testing.B) {
		b.StopTimer()
		rows, err := pqDB.Query("INSERT INTO bench_models (val) SELECT 'del' FROM generate_series(1, $1::int) RETURNING id", b.N)
		if err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		ids := make([]int64, 0, b.N)
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				b.Fatalf("scan id: %v", err)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		if err := rows.Close(); err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		b.StartTimer()
		for _, id := range ids {
			if _, err := pqDB.Exec("DELETE FROM bench_models WHERE id = $1", id); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm-delete", func(b *testing.B) {
		b.StopTimer()
		rows, err := pqDB.Query("INSERT INTO bench_models (val) SELECT 'del' FROM generate_series(1, $1::int) RETURNING id", b.N)
		if err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		ids := make([]int, 0, b.N)
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				b.Fatalf("scan id: %v", err)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		if err := rows.Close(); err != nil {
			b.Fatalf("insert for delete: %v", err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		b.StartTimer()
		for _, id := range ids {
			if err := gormDB.Delete(&BenchModel{}, id).Error; err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDrivers(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		prePull(b, rt, imagePostgres)
		connStr, teardown := setupPostgres(b, rt)
		defer teardown()
		runDriverBenchmarks(b, connStr)
	})
}
