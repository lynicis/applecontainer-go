package benchmarks

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/moby/moby/api/types/network"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkImagePull(b *testing.B) {
	if os.Getenv("APPLECONTAINER_BENCHMARK") == "" {
		b.Fatal("Set APPLECONTAINER_BENCHMARK=1 to run benchmarks")
	}

	b.Run("testcontainers-go", func(b *testing.B) {
		b.ReportAllocs()
		SkipIfDockerNotHealthy(b)
		img := "postgres:alpine"
		ctx := context.Background()
		prePull(b, TestcontainersGo, img)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:        img,
					ExposedPorts: []string{"5432"},
					Env: map[string]string{
						"POSTGRES_PASSWORD": "test",
					},
					WaitingFor: tcwait.ForSQL("5432", "pgx", func(host string, port network.Port) string {
						return "postgres://postgres:test@" + host + ":" + fmt.Sprint(port) + "/postgres?sslmode=disable"
					}).WithStartupTimeout(300 * time.Second),
				},
			}
			c, err := tccontainer.GenericContainer(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
			b.StopTimer()
			_ = c.Terminate(ctx)
			b.StartTimer()
		}
	})

	b.Run("applecontainer", func(b *testing.B) {
		b.ReportAllocs()
		SkipIfProviderNotHealthy(b)
		img := "postgres:alpine"
		ctx := context.Background()
		prePull(b, AppleContainer, img)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
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
			b.StopTimer()
			_ = c.Terminate(ctx)
			b.StartTimer()
		}
	})
}

func BenchmarkImageBuild(b *testing.B) {
	if os.Getenv("APPLECONTAINER_BENCHMARK") == "" {
		b.Fatal("Set APPLECONTAINER_BENCHMARK=1 to run benchmarks")
	}

	b.Run("testcontainers-go", func(b *testing.B) {
		b.ReportAllocs()
		SkipIfDockerNotHealthy(b)
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
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
			b.StopTimer()
			_ = c.Terminate(ctx)
			b.StartTimer()
		}
	})
}
