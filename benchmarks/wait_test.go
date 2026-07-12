package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkWaitStrategyHTTP(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				b.StartTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForHTTP("/").WithPort("80")).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		case TestcontainersGo:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				req := tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        img,
						ExposedPorts: []string{"80"},
						WaitingFor:   tcwait.ForHTTP("/").WithPort("80"),
					},
					Started: true,
				}
				b.StartTimer()
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}

func BenchmarkWaitStrategySQL(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "postgres:alpine"
		env := map[string]string{"POSTGRES_PASSWORD": "test"}

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				b.StartTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithExposedPorts("5432"),
					applecontainer.WithEnv(env),
					applecontainer.WithWaitingFor(wait.ForAll(
						wait.ForSQL("5432", "pgx", func(host string, port int) string {
							return fmt.Sprintf("user=postgres password=test host=%s port=%d dbname=postgres sslmode=disable", host, port)
						})).WithDeadline(

						120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		case TestcontainersGo:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				req := tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        img,
						ExposedPorts: []string{"5432/tcp"},
						Env:          env,
						WaitingFor:   tcwait.ForLog("database system is ready to accept connections").WithStartupTimeout(120 * time.Second),
					},
					Started: true,
				}
				b.StartTimer()
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}

func BenchmarkWaitStrategyExec(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "alpine:latest"
		cmd := []string{"sh", "-c", "sleep 3 && true; sleep 3600"}

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				b.StartTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithCmd(cmd...),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForExec([]string{"true"})).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		case TestcontainersGo:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				req := tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:      img,
						Cmd:        cmd,
						WaitingFor: tcwait.ForExec([]string{"true"}),
					},
					Started: true,
				}
				b.StartTimer()
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}

func BenchmarkWaitStrategyHealth(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"
		cmd := []string{"nginx", "-g", "daemon off;"}

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				b.StartTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithCmd(cmd...),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForHealth()).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		case TestcontainersGo:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				req := tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        img,
						ExposedPorts: []string{"80/tcp"},
						Cmd:          cmd,
						// tcwait.ForHealthCheck() requires the container to have a HEALTHCHECK instruction.
						// We'll emulate it by waiting for state "running", which has similar daemon-polling overhead.
						WaitingFor: tcwait.ForListeningPort("80/tcp").WithStartupTimeout(120 * time.Second), // proxy for daemon fast-return
					},
					Started: true,
				}
				b.StartTimer()
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}

func BenchmarkWaitStrategyComposite(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				b.StartTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitingFor(wait.ForAll(
						wait.ForAll(
							wait.ForLog("ready for start up"),
							wait.ForHTTP("/").WithPort("80"),
						)).WithDeadline(

						120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		case TestcontainersGo:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				req := tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        img,
						ExposedPorts: []string{"80"},
						WaitingFor: tcwait.ForAll(
							tcwait.ForLog("ready for start up"),
							tcwait.ForHTTP("/").WithPort("80"),
						),
					},
					Started: true,
				}
				b.StartTimer()
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}
