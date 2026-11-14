package benchmarks

import (
	"context"
	"io"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

// nginxRequest returns a testcontainers request for nginx, matching the Apple
// side's log wait strategy.
func nginxRequest() tccontainer.GenericContainerRequest {
	return tccontainer.GenericContainerRequest{
		ContainerRequest: tccontainer.ContainerRequest{
			Image:        imageNginx,
			ExposedPorts: []string{"80"},
			WaitingFor:   tcwait.ForLog("ready for start up"),
		},
		Started: true,
	}
}

func BenchmarkStop(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		stopTimeout := 5 * time.Second

		switch rt {
		case AppleContainer:
			for range b.N {
				b.StopTimer()
				c, err := applecontainer.Run(ctx, imageNginx,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				err = c.Stop(ctx, &stopTimeout)
				b.StopTimer()
				if err != nil {
					b.Fatal(err)
				}
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		case TestcontainersGo:
			for range b.N {
				b.StopTimer()
				c, err := tccontainer.GenericContainer(ctx, nginxRequest())
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				err = c.Stop(ctx, &stopTimeout)
				b.StopTimer()
				if err != nil {
					b.Fatal(err)
				}
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

func BenchmarkStart(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		stopTimeout := 5 * time.Second

		switch rt {
		case AppleContainer:
			for range b.N {
				b.StopTimer()
				c, err := applecontainer.Run(ctx, imageNginx,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := c.Stop(ctx, &stopTimeout); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				err = c.Start(ctx)
				b.StopTimer()
				if err != nil {
					b.Fatal(err)
				}
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		case TestcontainersGo:
			for range b.N {
				b.StopTimer()
				c, err := tccontainer.GenericContainer(ctx, nginxRequest())
				if err != nil {
					b.Fatal(err)
				}
				if err := c.Stop(ctx, &stopTimeout); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				err = c.Start(ctx)
				b.StopTimer()
				if err != nil {
					b.Fatal(err)
				}
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

func BenchmarkTerminate(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()

		switch rt {
		case AppleContainer:
			for range b.N {
				b.StopTimer()
				c, err := applecontainer.Run(ctx, imageNginx,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		case TestcontainersGo:
			for range b.N {
				b.StopTimer()
				c, err := tccontainer.GenericContainer(ctx, nginxRequest())
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

func BenchmarkInspect(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, imageNginx,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				if _, err := c.Inspect(ctx); err != nil {
					b.Fatal(err)
				}
			}
		case TestcontainersGo:
			b.StopTimer()
			c, err := tccontainer.GenericContainer(ctx, nginxRequest())
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				if _, err := c.Inspect(ctx); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

func BenchmarkExec(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, imageNginx,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				if _, _, err := c.Exec(ctx, []string{"echo", "hello"}); err != nil {
					b.Fatal(err)
				}
			}
		case TestcontainersGo:
			b.StopTimer()
			c, err := tccontainer.GenericContainer(ctx, nginxRequest())
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				code, reader, err := c.Exec(ctx, []string{"echo", "hello"})
				if err != nil {
					b.Fatal(err)
				}
				if _, err := io.ReadAll(reader); err != nil {
					b.Fatal(err)
				}
				if code != 0 {
					b.Fatalf("exec exit code: %d", code)
				}
			}
		}
	})
}

func copyBenchmark(b *testing.B, rt Runtime, data []byte) {
	ctx := context.Background()

	switch rt {
	case AppleContainer:
		b.StopTimer()
		c, err := applecontainer.Run(ctx, imageNginx,
			applecontainer.WithExposedPorts("80"),
			applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
		)
		if err != nil {
			b.Fatal(err)
		}
		defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
		b.StartTimer()

		for range b.N {
			if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
				b.Fatal(err)
			}
		}
	case TestcontainersGo:
		b.StopTimer()
		c, err := tccontainer.GenericContainer(ctx, nginxRequest())
		if err != nil {
			b.Fatal(err)
		}
		defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
		b.StartTimer()

		for range b.N {
			if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkCopyFile1KB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		copyBenchmark(b, rt, make([]byte, 1024))
	})
}

func BenchmarkCopyFile1MB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		copyBenchmark(b, rt, make([]byte, 1024*1024))
	})
}

func copyFromContainerBenchmark(b *testing.B, rt Runtime, data []byte) {
	ctx := context.Background()

	switch rt {
	case AppleContainer:
		b.StopTimer()
		c, err := applecontainer.Run(ctx, imageNginx,
			applecontainer.WithExposedPorts("80"),
			applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
		)
		if err != nil {
			b.Fatal(err)
		}
		defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()

		if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
			b.Fatal(err)
		}

		b.StartTimer()

		for range b.N {
			rc, err := c.CopyFileFromContainer(ctx, "/tmp/payload")
			if err != nil {
				b.Fatal(err)
			}
			if _, err := io.Copy(io.Discard, rc); err != nil {
				b.Fatal(err)
			}
			if err := rc.Close(); err != nil {
				b.Fatal(err)
			}
		}
	case TestcontainersGo:
		b.StopTimer()
		c, err := tccontainer.GenericContainer(ctx, nginxRequest())
		if err != nil {
			b.Fatal(err)
		}
		defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()

		if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
			b.Fatal(err)
		}

		b.StartTimer()

		for range b.N {
			rc, err := c.CopyFileFromContainer(ctx, "/tmp/payload")
			if err != nil {
				b.Fatal(err)
			}
			if _, err := io.Copy(io.Discard, rc); err != nil {
				b.Fatal(err)
			}
			if err := rc.Close(); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkCopyFileFromContainer1KB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		copyFromContainerBenchmark(b, rt, make([]byte, 1024))
	})
}

func BenchmarkCopyFileFromContainer1MB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		copyFromContainerBenchmark(b, rt, make([]byte, 1024*1024))
	})
}

func BenchmarkLogs(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		// Alpine printing ~10k lines of logs quickly
		img := imageAlpine
		cmd := []string{"sh", "-c", "for i in $(seq 1 10000); do echo 'log line output testing performance' $i; done; sleep 3600"}

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithCmd(cmd...),
				applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("log line output testing performance 10000")).WithDeadline(120*time.Second)),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				rc, err := c.Logs(ctx)
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		case TestcontainersGo:
			b.StopTimer()
			c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:      img,
					Cmd:        cmd,
					WaitingFor: tcwait.ForLog("log line output testing performance 10000"),
				},
				Started: true,
			})
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				rc, err := c.Logs(ctx)
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		}
	})
}

func BenchmarkWaitStrategyLog(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := imageAlpine
		// Print 5000 lines then ready
		cmd := []string{"sh", "-c", "for i in $(seq 1 5000); do echo 'spam line' $i; done; echo 'READY'; sleep 3600"}

		switch rt {
		case AppleContainer:
			for range b.N {
				b.StopTimer()
				// Note: we measure the ENTIRE Run call which includes WaitStrategy execution
				b.StartTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithCmd(cmd...),
					applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("READY")).WithDeadline(120*time.Second)),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		case TestcontainersGo:
			for range b.N {
				b.StopTimer()
				b.StartTimer()
				c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
					ContainerRequest: tccontainer.ContainerRequest{
						Image:      img,
						Cmd:        cmd,
						WaitingFor: tcwait.ForLog("READY"),
					},
					Started: true,
				})
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				if err := c.Terminate(ctx); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}
