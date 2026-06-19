package benchmarks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"

	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

func BenchmarkHTTPThroughput(b *testing.B) {
	// Apple container networking doesn't support inbound HTTP from host.
	// Only benchmark testcontainers-go for HTTP throughput.
	b.Run("TestcontainersGo", func(b *testing.B) {
		ctx := context.Background()
		img := "nginx:alpine"
		prePull(b, TestcontainersGo, img)

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
		defer c.Terminate(ctx)
		host, _ := c.Host(ctx)
		port, _ := c.MappedPort(ctx, "80")
		endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())
		b.StartTimer()

		client := &http.Client{Timeout: 5 * time.Second}
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(endpoint)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

func BenchmarkTCPLatency(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := "redis:alpine"

		b.StopTimer()
		switch rt {
		case AppleContainer:
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("6379"),
				applecontainer.WithWaitStrategyAndDeadline(wait.ForLog("Ready"), 120*time.Second),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer c.Terminate(ctx)
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_, output, err := c.Exec(ctx, []string{"redis-cli", "ping"})
				if err != nil {
					b.Fatal(err)
				}
				if !strings.Contains(string(output), "PONG") {
					b.Fatalf("expected PONG, got: %s", string(output))
				}
			}
		case TestcontainersGo:
			req := tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:        img,
					ExposedPorts: []string{"6379"},
					WaitingFor:   tcwait.ForLog("Ready"),
				},
				Started: true,
			}
			c, err := tccontainer.GenericContainer(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
			defer c.Terminate(ctx)
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				code, reader, err := c.Exec(ctx, []string{"redis-cli", "ping"})
				if err != nil {
					b.Fatal(err)
				}
				output, _ := io.ReadAll(reader)
				if !strings.Contains(string(output), "PONG") {
					b.Fatalf("expected PONG, got: %s", string(output))
				}
				_ = code
			}
		}
	})
}
