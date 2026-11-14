package benchmarks

import (
	"context"
	"testing"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	"github.com/moby/moby/client"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

// BenchmarkImageBuild measures an uncached build of Dockerfile.bench plus
// starting the resulting image, on both runtimes.
func BenchmarkImageBuild(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()

		for range b.N {
			var terminate func() error
			switch rt {
			case AppleContainer:
				c, err := applecontainer.Run(ctx, "",
					applecontainer.WithContainerfile(applecontainer.FromContainerfile{
						Context: ".",
						File:    "Dockerfile.bench",
						NoCache: true,
					}),
					applecontainer.WithWaitingFor(wait.ForExit()),
				)
				if err != nil {
					b.Fatal(err)
				}
				terminate = func() error { return c.Terminate(ctx) }
			case TestcontainersGo:
				c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						FromDockerfile: tccontainer.FromDockerfile{
							Context:    ".",
							Dockerfile: "Dockerfile.bench",
							Repo:       "bench-alpine",
							Tag:        "latest",
							BuildOptionsModifier: func(opts *client.ImageBuildOptions) {
								opts.NoCache = true
							},
						},
						WaitingFor: tcwait.ForExit(),
					},
					Started: true,
				})
				if err != nil {
					b.Fatal(err)
				}
				terminate = func() error { return c.Terminate(ctx) }
			}
			b.StopTimer()
			if err := terminate(); err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
		}
	})
}
