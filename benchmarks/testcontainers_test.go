package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkTestcontainers(b *testing.B) {
	ctx := context.Background()

	b.Run("postgres", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			t0 := time.Now()
			pgContainer, _ := postgres.Run(ctx,
				"postgres:15-alpine",
				postgres.WithDatabase("bench"),
				postgres.WithUsername("bench"),
				postgres.WithPassword("bench"),
				testcontainers.WithWaitStrategy(
					wait.ForLog("database system is ready to accept connections").
						WithOccurrence(2).WithStartupTimeout(5*time.Second),
				),
			)
			tReady := time.Now()

			if pgContainer != nil {
				_ = pgContainer.Terminate(ctx)
			}
			tTeardown := time.Now()

			b.ReportMetric(float64(tReady.Sub(t0).Milliseconds()), "startup+ready_ms/op")
			b.ReportMetric(float64(tTeardown.Sub(tReady).Milliseconds()), "teardown_ms/op")
		}
	})

	b.Run("redis", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			t0 := time.Now()
			redisContainer, _ := redis.Run(ctx, "redis:alpine")
			tReady := time.Now()

			if redisContainer != nil {
				_ = redisContainer.Terminate(ctx)
			}
			tTeardown := time.Now()

			b.ReportMetric(float64(tReady.Sub(t0).Milliseconds()), "startup+ready_ms/op")
			b.ReportMetric(float64(tTeardown.Sub(tReady).Milliseconds()), "teardown_ms/op")
		}
	})

	b.Run("nginx", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			t0 := time.Now()
			req := testcontainers.ContainerRequest{
				Image:        "nginx:alpine",
				ExposedPorts: []string{"80/tcp"},
				WaitingFor:   wait.ForHTTP("/"),
			}
			nginxContainer, _ := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			})
			tReady := time.Now()

			if nginxContainer != nil {
				_ = nginxContainer.Terminate(ctx)
			}
			tTeardown := time.Now()

			b.ReportMetric(float64(tReady.Sub(t0).Milliseconds()), "startup+ready_ms/op")
			b.ReportMetric(float64(tTeardown.Sub(tReady).Milliseconds()), "teardown_ms/op")
		}
	})
}
