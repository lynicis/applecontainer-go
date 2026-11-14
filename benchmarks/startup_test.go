package benchmarks

import (
	"context"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

type startupCase struct {
	name   string
	image  string
	ports  []string
	env    map[string]string
	wait   wait.Strategy
	tcWait tcwait.Strategy
}

func startupCases() []startupCase {
	return []startupCase{
		{
			name:   "nginx",
			image:  imageNginx,
			ports:  []string{"80"},
			wait:   wait.ForLog("ready for start up"),
			tcWait: tcwait.ForLog("ready for start up").WithStartupTimeout(120 * time.Second),
		},
		{
			name:   "redis",
			image:  imageRedis,
			ports:  []string{"6379"},
			wait:   wait.ForLog("Ready"),
			tcWait: tcwait.ForLog("Ready").WithStartupTimeout(120 * time.Second),
		},
		{
			name:   "postgres",
			image:  imagePostgres,
			ports:  []string{"5432"},
			env:    map[string]string{"POSTGRES_PASSWORD": "bench"},
			wait:   wait.ForExec([]string{"pg_isready", "-U", "postgres"}),
			tcWait: tcwait.ForExec([]string{"pg_isready", "-U", "postgres"}).WithStartupTimeout(120 * time.Second),
		},
	}
}

// BenchmarkStartup measures create+start+ready for the same image and wait
// strategy on both runtimes.
func BenchmarkStartup(b *testing.B) {
	for _, c := range startupCases() {
		b.Run(c.name, func(b *testing.B) {
			RunWithBoth(b, func(b *testing.B, rt Runtime) {
				ctx := context.Background()
				prePull(b, rt, c.image)
				b.ResetTimer()

				for range b.N {
					var terminate func()
					switch rt {
					case AppleContainer:
						cont, err := applecontainer.Run(ctx, c.image,
							applecontainer.WithExposedPorts(c.ports...),
							applecontainer.WithEnv(c.env),
							applecontainer.WithWaitingFor(wait.ForAll(c.wait).WithDeadline(120*time.Second)),
						)
						if err != nil {
							b.Fatal(err)
						}
						terminate = func() { _ = cont.Terminate(ctx) }
					case TestcontainersGo:
						cont, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
							ContainerRequest: tccontainer.ContainerRequest{
								Image:        c.image,
								ExposedPorts: c.ports,
								Env:          c.env,
								WaitingFor:   c.tcWait,
							},
							Started: true,
						})
						if err != nil {
							b.Fatal(err)
						}
						terminate = func() { _ = cont.Terminate(ctx) }
					}
					b.StopTimer()
					terminate()
					b.StartTimer()
				}
			})
		})
	}
}
