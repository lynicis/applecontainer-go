package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	"github.com/moby/moby/api/types/network"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkWaitStrategyHTTP(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()

		switch rt {
		case AppleContainer:
			for range b.N {
				c, err := applecontainer.Run(ctx, imageNginx,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForHTTP("/").WithPort("80")).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		case TestcontainersGo:
			for range b.N {
				c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        imageNginx,
						ExposedPorts: []string{"80"},
						WaitingFor:   tcwait.ForHTTP("/").WithPort("80"),
					},
					Started: true,
				})
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		}
	})
}

func BenchmarkWaitStrategySQL(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		env := map[string]string{"POSTGRES_PASSWORD": "test"}

		switch rt {
		case AppleContainer:
			for range b.N {
				c, err := applecontainer.Run(ctx, imagePostgres,
					applecontainer.WithExposedPorts("5432"),
					applecontainer.WithEnv(env),
					applecontainer.WithWaitingFor(wait.ForAll(
						wait.ForSQL("5432", "pgx", func(host string, port int) string {
							return fmt.Sprintf("postgres://postgres:test@%s:%d/postgres?sslmode=disable", host, port)
						})).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		case TestcontainersGo:
			for range b.N {
				c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        imagePostgres,
						ExposedPorts: []string{"5432/tcp"},
						Env:          env,
						WaitingFor: tcwait.ForSQL("5432", "pgx", func(host string, port network.Port) string {
							return fmt.Sprintf("postgres://postgres:test@%s:%s/postgres?sslmode=disable", host, port.Port())
						}).WithStartupTimeout(120 * time.Second),
					},
					Started: true,
				})
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		}
	})
}

func BenchmarkWaitStrategyExec(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		cmd := []string{"sh", "-c", "sleep 3 && true; sleep 3600"}

		switch rt {
		case AppleContainer:
			for range b.N {
				c, err := applecontainer.Run(ctx, imageAlpine,
					applecontainer.WithCmd(cmd...),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForExec([]string{"true"})).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		case TestcontainersGo:
			for range b.N {
				c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:      imageAlpine,
						Cmd:        cmd,
						WaitingFor: tcwait.ForExec([]string{"true"}),
					},
					Started: true,
				})
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		}
	})
}

func BenchmarkWaitStrategyHealth(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		cmd := []string{"nginx", "-g", "daemon off;"}

		switch rt {
		case AppleContainer:
			for range b.N {
				c, err := applecontainer.Run(ctx, imageNginx,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithCmd(cmd...),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForHealth()).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		case TestcontainersGo:
			for range b.N {
				c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        imageNginx,
						ExposedPorts: []string{"80/tcp"},
						Cmd:          cmd,
						// The two runtimes have no common health primitive: tcwait.ForHealthCheck()
						// requires a HEALTHCHECK instruction, and nginx:alpine has none. The delta
						// measured here is state-polling (Apple ForHealth) versus port-polling
						// (ForListeningPort), not health-check parity.
						WaitingFor: tcwait.ForListeningPort("80/tcp").WithStartupTimeout(120 * time.Second),
					},
					Started: true,
				})
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		}
	})
}

func BenchmarkWaitStrategyComposite(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()

		switch rt {
		case AppleContainer:
			for range b.N {
				c, err := applecontainer.Run(ctx, imageNginx,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitingFor(wait.ForAll(
						wait.ForAll(
							wait.ForLog("ready for start up"),
							wait.ForHTTP("/").WithPort("80"),
						)).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		case TestcontainersGo:
			for range b.N {
				c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:        imageNginx,
						ExposedPorts: []string{"80"},
						WaitingFor: tcwait.ForAll(
							tcwait.ForLog("ready for start up"),
							tcwait.ForHTTP("/").WithPort("80"),
						),
					},
					Started: true,
				})
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = c.Terminate(ctx)
				b.StartTimer()
			}
		}
	})
}
