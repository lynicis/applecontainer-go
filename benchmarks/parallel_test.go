package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkParallel(b *testing.B) {
	for _, n := range []int{2, 4, 8} {
		b.Run(fmt.Sprintf("containers=%d", n), func(b *testing.B) {
			RunWithBoth(b, func(b *testing.B, rt Runtime) {
				img := imageNginx
				prePull(b, rt, img)
				b.ResetTimer()

				for range b.N {
					ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
					var wg sync.WaitGroup
					var mu sync.Mutex
					var cleanups []func()
					var firstErr error

					for range n {
						wg.Add(1)
						go func() {
							defer wg.Done()
							switch rt {
							case AppleContainer:
								c, err := applecontainer.Run(ctx, img,
									applecontainer.WithExposedPorts("80"),
									applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
								)
								if err != nil {
									mu.Lock()
									if firstErr == nil {
										firstErr = err
									}
									mu.Unlock()
									return
								}
								mu.Lock()
								cleanups = append(cleanups, func() { _ = c.Terminate(ctx) })
								mu.Unlock()
							case TestcontainersGo:
								c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
									ContainerRequest: tccontainer.ContainerRequest{
										Image:        img,
										ExposedPorts: []string{"80"},
										WaitingFor:   tcwait.ForLog("ready for start up"),
									},
									Started: true,
								})
								if err != nil {
									mu.Lock()
									if firstErr == nil {
										firstErr = err
									}
									mu.Unlock()
									return
								}
								mu.Lock()
								cleanups = append(cleanups, func() { _ = c.Terminate(ctx) })
								mu.Unlock()
							}
						}()
					}
					wg.Wait()
					b.StopTimer()
					for _, fn := range cleanups {
						fn()
					}
					cancel()
					if firstErr != nil {
						b.Fatal(firstErr)
					}
					b.StartTimer()
				}
			})
		})
	}
}
