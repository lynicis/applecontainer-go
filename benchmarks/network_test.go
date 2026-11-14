package benchmarks

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
	tccontainer "github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

// BenchmarkHTTPThroughput measures end-to-end HTTP GET throughput against nginx.
func BenchmarkHTTPThroughput(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		prePull(b, rt, imageNginx)

		b.StopTimer()
		var endpoint string
		var terminate func()
		switch rt {
		case AppleContainer:
			c, err := applecontainer.Run(ctx, imageNginx,
				applecontainer.WithExposedPorts("80"),
				applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("ready for start up")).WithDeadline(120*time.Second)),
			)
			if err != nil {
				b.Fatal(err)
			}
			terminate = func() { _ = c.Terminate(ctx) }
			ep, err := c.Endpoint(ctx, "80")
			if err != nil {
				b.Fatal(err)
			}
			endpoint = "http://" + ep
		case TestcontainersGo:
			c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:        imageNginx,
					ExposedPorts: []string{"80"},
					WaitingFor:   tcwait.ForLog("ready for start up"),
				},
				Started: true,
			})
			if err != nil {
				b.Fatal(err)
			}
			terminate = func() { _ = c.Terminate(ctx) }
			host, err := c.Host(ctx)
			if err != nil {
				b.Fatal(err)
			}
			port, err := c.MappedPort(ctx, "80")
			if err != nil {
				b.Fatal(err)
			}
			endpoint = "http://" + net.JoinHostPort(host, port.Port())
		}
		defer func() { b.StopTimer(); terminate() }()

		client := &http.Client{Timeout: 5 * time.Second}

		resp, err := client.Get(endpoint)
		if err != nil {
			b.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		b.SetBytes(int64(len(body)))
		b.StartTimer()

		for range b.N {
			resp, err := client.Get(endpoint)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})
}

// BenchmarkExecRedisPing measures `Exec redis-cli ping` on both runtimes.
func BenchmarkExecRedisPing(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		img := imageRedis

		b.StopTimer()
		switch rt {
		case AppleContainer:
			c, err := applecontainer.Run(ctx, img,
				applecontainer.WithExposedPorts("6379"),
				applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("Ready")).WithDeadline(120*time.Second)),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				_, output, err := c.Exec(ctx, []string{"redis-cli", "ping"})
				if err != nil {
					b.Fatal(err)
				}
				if !strings.Contains(string(output), "PONG") {
					b.Fatalf("expected PONG, got: %s", string(output))
				}
			}
		case TestcontainersGo:
			c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:        img,
					ExposedPorts: []string{"6379"},
					WaitingFor:   tcwait.ForLog("Ready"),
				},
				Started: true,
			})
			if err != nil {
				b.Fatal(err)
			}
			defer func() { b.StopTimer(); _ = c.Terminate(ctx) }()
			b.StartTimer()

			for range b.N {
				_, reader, err := c.Exec(ctx, []string{"redis-cli", "ping"})
				if err != nil {
					b.Fatal(err)
				}
				output, _ := io.ReadAll(reader)
				if !strings.Contains(string(output), "PONG") {
					b.Fatalf("expected PONG, got: %s", string(output))
				}
			}
		}
	})
}

// BenchmarkTCPRoundTrip measures direct TCP PING/PONG round trips against Redis.
func BenchmarkTCPRoundTrip(b *testing.B) {
	RunWithBoth(b, func(b *testing.B, rt Runtime) {
		ctx := context.Background()
		prePull(b, rt, imageRedis)

		b.StopTimer()
		var endpoint string
		var terminate func()
		switch rt {
		case AppleContainer:
			c, err := applecontainer.Run(ctx, imageRedis,
				applecontainer.WithExposedPorts("6379"),
				applecontainer.WithWaitingFor(wait.ForAll(wait.ForLog("Ready")).WithDeadline(120*time.Second)),
			)
			if err != nil {
				b.Fatal(err)
			}
			terminate = func() { _ = c.Terminate(ctx) }
			ep, err := c.Endpoint(ctx, "6379")
			if err != nil {
				b.Fatal(err)
			}
			endpoint = ep
		case TestcontainersGo:
			c, err := tccontainer.GenericContainer(ctx, tccontainer.GenericContainerRequest{
				ContainerRequest: tccontainer.ContainerRequest{
					Image:        imageRedis,
					ExposedPorts: []string{"6379"},
					WaitingFor:   tcwait.ForLog("Ready"),
				},
				Started: true,
			})
			if err != nil {
				b.Fatal(err)
			}
			terminate = func() { _ = c.Terminate(ctx) }
			host, err := c.Host(ctx)
			if err != nil {
				b.Fatal(err)
			}
			port, err := c.MappedPort(ctx, "6379")
			if err != nil {
				b.Fatal(err)
			}
			endpoint = net.JoinHostPort(host, port.Port())
		}
		defer func() { b.StopTimer(); terminate() }()

		conn, err := net.Dial("tcp", endpoint)
		if err != nil {
			b.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		reader := bufio.NewReader(conn)

		if _, err := conn.Write([]byte("PING\r\n")); err != nil {
			b.Fatal(err)
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			b.Fatal(err)
		}
		if line != "+PONG\r\n" {
			b.Fatalf("expected +PONG, got: %q", line)
		}

		b.ResetTimer()
		b.StartTimer()

		for range b.N {
			if _, err := conn.Write([]byte("PING\r\n")); err != nil {
				b.Fatal(err)
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				b.Fatal(err)
			}
			if line != "+PONG\r\n" {
				b.Fatalf("expected +PONG, got: %q", line)
			}
		}
	})
}
