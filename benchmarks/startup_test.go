package benchmarks

import (
	"context"
	"os"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
)

type startupImage struct {
	label string
	image string
	wait  wait.Strategy
}

func startupImages() []startupImage {
	return []startupImage{
		{"nginx:alpine", "nginx:alpine", wait.ForLog("ready for start up")},
		{"redis:alpine", "redis:alpine", wait.ForLog("Ready")},
		{"postgres:alpine", "postgres:alpine", wait.ForExec([]string{"pg_isready", "-U", "postgres"})},
	}
}

func BenchmarkStartup(b *testing.B) {
	if os.Getenv("APPLECONTAINER_BENCHMARK") == "" {
		b.Fatal("Set APPLECONTAINER_BENCHMARK=1 to run benchmarks")
	}

	for _, img := range startupImages() {
		b.Run(img.label, func(b *testing.B) {
			SkipIfProviderNotHealthy(b)
			ctx := context.Background()
			prePull(b, AppleContainer, img.image)

			for i := 0; i < b.N; i++ {
				c, err := applecontainer.Run(ctx, img.image,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithEnv(map[string]string{"POSTGRES_PASSWORD": "test"}),
					applecontainer.WithWaitStrategyAndDeadline(img.wait, 120*time.Second),
				)
				if err != nil {
					b.Fatal(err)
				}
				c.Terminate(ctx)
			}
		})
	}
}
