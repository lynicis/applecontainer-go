package benchmarks

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"

	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkStop(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"
		stopTimeout := 5 * time.Second

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				_ = c.Stop(ctx, &stopTimeout)
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
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				_ = c.Stop(ctx, &stopTimeout)
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}

func BenchmarkStart(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"
		stopTimeout := 5 * time.Second

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
				)
				if err != nil {
					b.Fatal(err)
				}
				_ = c.Stop(ctx, &stopTimeout)
				b.StartTimer()
				_ = c.Start(ctx)
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
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				_ = c.Stop(ctx, &stopTimeout)
				b.StartTimer()
				_ = c.Start(ctx)
				b.StopTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}

func BenchmarkTerminate(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithExposedPorts("80"),
					applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
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
				c, err := tccontainer.GenericContainer(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				_ = c.Terminate(ctx)
			}
		}
	})
}

func BenchmarkInspect(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_, _ = c.Inspect(ctx)
			}
		case TestcontainersGo:
			b.StopTimer()
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
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_, _ = c.Inspect(ctx)
			}
		}
	})
}

func BenchmarkExec(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_, _, _ = c.Exec(ctx, []string{"echo", "hello"})
			}
		case TestcontainersGo:
			b.StopTimer()
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
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				code, reader, err := c.Exec(ctx, []string{"echo", "hello"})
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.ReadAll(reader)
				_ = code
			}
		}
	})
}

func BenchmarkCopyFile1KB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		tmpDir := b.TempDir()
		srcPath := filepath.Join(tmpDir, "payload")
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if err := os.WriteFile(srcPath, data, 0o644); err != nil {
			b.Fatal(err)
		}

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = c.CopyToContainer(ctx, data, "/tmp/payload", 0o644)
			}
		case TestcontainersGo:
			b.StopTimer()
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
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = c.CopyToContainer(ctx, data, "/tmp/payload", 0o644)
			}
		}
	})
}

func BenchmarkCopyFile1MB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		tmpDir := b.TempDir()
		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		srcPath := filepath.Join(tmpDir, "payload")
		if err := os.WriteFile(srcPath, data, 0o644); err != nil {
			b.Fatal(err)
		}

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = c.CopyToContainer(ctx, data, "/tmp/payload", 0o644)
			}
		case TestcontainersGo:
			b.StopTimer()
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
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = c.CopyToContainer(ctx, data, "/tmp/payload", 0o644)
			}
		}
	})
}

func BenchmarkCopyFileFromContainer1KB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()

			if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
				b.Fatal(err)
			}

			b.StartTimer()

			for i := 0; i < b.N; i++ {
				rc, err := c.CopyFileFromContainer(ctx, "/tmp/payload")
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		case TestcontainersGo:
			b.StopTimer()
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
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()

			if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
				b.Fatal(err)
			}

			b.StartTimer()

			for i := 0; i < b.N; i++ {
				rc, err := c.CopyFileFromContainer(ctx, "/tmp/payload")
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		}
	})
}

func BenchmarkCopyFileFromContainer1MB(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "nginx:alpine"

		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("ready for start up"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()

			if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
				b.Fatal(err)
			}

			b.StartTimer()

			for i := 0; i < b.N; i++ {
				rc, err := c.CopyFileFromContainer(ctx, "/tmp/payload")
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		case TestcontainersGo:
			b.StopTimer()
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
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()

			if err := c.CopyToContainer(ctx, data, "/tmp/payload", 0o644); err != nil {
				b.Fatal(err)
			}

			b.StartTimer()

			for i := 0; i < b.N; i++ {
				rc, err := c.CopyFileFromContainer(ctx, "/tmp/payload")
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		}
	})
}

func BenchmarkLogs(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		// Alpine printing ~10k lines of logs quickly
		img := "alpine:latest"
		cmd := []string{"sh", "-c", "for i in $(seq 1 10000); do echo 'log line output testing performance' $i; done; sleep 3600"}

		switch rt {
		case AppleContainer:
			b.StopTimer()
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithCmd(cmd...),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("log line output testing performance 10000"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				rc, err := c.Logs(ctx)
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		case TestcontainersGo:
			b.StopTimer()
			req := tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:      img,
					Cmd:        cmd,
					WaitingFor: tcwait.ForLog("log line output testing performance 10000"),
				},
				Started: true,
			}
			c, err := tccontainer.GenericContainer(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = c.Terminate(ctx) }()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
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
		img := "alpine:latest"
		// Print 5000 lines then ready
		cmd := []string{"sh", "-c", "for i in $(seq 1 5000); do echo 'spam line' $i; done; echo 'READY'; sleep 3600"}

		switch rt {
		case AppleContainer:
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Note: we measure the ENTIRE Run call which includes WaitStrategy execution
				b.StartTimer()
				c, err := applecontainer.Run(ctx, img,
					applecontainer.WithCmd(cmd...),
					applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("READY"), 120*time.Second),
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
						WaitingFor: tcwait.ForLog("READY"),
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
