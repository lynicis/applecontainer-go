package benchmarks

import (
	"context"
	"os"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkImagePull(b *testing.B) {
	if os.Getenv("APPLECONTAINER_BENCHMARK") == "" {
		b.Fatal("Set APPLECONTAINER_BENCHMARK=1 to run benchmarks")
	}

	b.Run("testcontainers-go", func(b *testing.B) {
		SkipIfDockerNotHealthy(b)
		img := "postgres:alpine"
		ctx := context.Background()
		prePull(b, TestcontainersGo, img)

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			req := tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:        img,
					ExposedPorts: []string{"5432"},
					Env: map[string]string{
						"POSTGRES_PASSWORD": "test",
					},
					WaitingFor: tcwait.ForSQL("5432", "pgx", func(host, port string) string {
						return "postgres://postgres:test@" + host + ":" + port + "/postgres?sslmode=disable"
					}).WithStartupTimeout(300 * time.Second),
				},
			}
			c, err := tccontainer.GenericContainer(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			c.Terminate(ctx)
		}
	})

	b.Run("applecontainer", func(b *testing.B) {
		SkipIfProviderNotHealthy(b)
		img := "postgres:alpine"
		ctx := context.Background()
		prePull(b, AppleContainer, img)

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("5432"),
				applecontainer.WithEnv(map[string]string{"POSTGRES_PASSWORD": "test"}),
				applecontainer.WithWaitStrategyAndDeadline(
					wait.ForExec([]string{"pg_isready", "-U", "postgres"}),
					300*time.Second,
				),
			)
			if err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			c.Terminate(ctx)
		}
	})
}

func BenchmarkImageBuild(b *testing.B) {
	if os.Getenv("APPLECONTAINER_BENCHMARK") == "" {
		b.Fatal("Set APPLECONTAINER_BENCHMARK=1 to run benchmarks")
	}

	b.Run("testcontainers-go", func(b *testing.B) {
		SkipIfDockerNotHealthy(b)
		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			req := tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					FromDockerfile: tccontainer.FromDockerfile{
						Dockerfile: "Dockerfile.bench",
						Context:    ".",
						Repo:       "bench-alpine",
						Tag:        "latest",
					},
					WaitingFor: tcwait.ForExit(),
				},
				Started: true,
			}
			c, err := tccontainer.GenericContainer(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			c.Terminate(ctx)
		}
	})
}
