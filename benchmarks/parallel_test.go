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
		n := n
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			RunWithBoth(b, func(b *testing.B, rt Runtime) {
				img := "nginx:alpine"
				prePull(b, rt, img)
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
					var wg sync.WaitGroup
					var mu sync.Mutex
					var cleanups []func()

					for j := 0; j < n; j++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							switch rt {
							case AppleContainer:
								c, err := applecontainer.Run(ctx, img,
									applecontainer.WithExposedPorts("80"),
									applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
								)
								if err != nil {
									b.Error(err)
									return
								}
								mu.Lock()
								cleanups = append(cleanups, func() { _ = c.Terminate(ctx) })
								mu.Unlock()
							case TestcontainersGo:
								req := tccontainer.GenericContainerRequest{
									ContainerRequest: tccontainer.ContainerRequest{
										Image:        img,
										ExposedPorts: []string{"80"},
										WaitingFor:   tcwait.ForHTTP("/").WithPort("80"),
									},
									Started: true,
								}
								c, err := tccontainer.GenericContainer(ctx, req)
								if err != nil {
									b.Error(err)
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
					b.StartTimer()
				}
			})
		})
	}
}
