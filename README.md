# applecontainer-go

A Go library that runs Linux containers as test dependencies through Apple's native `container` CLI, with a `testcontainers-go`-shaped API.

[![Go Reference](https://pkg.go.dev/badge/github.com/lynicis/applecontainer-go.svg)](https://pkg.go.dev/github.com/lynicis/applecontainer-go)
[![codecov](https://codecov.io/gh/lynicis/applecontainer-go/branch/main/graph/badge.svg)](https://codecov.io/gh/lynicis/applecontainer-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**Status:** experimental, pre-1.0 (latest release `v0.3.0`). The API is still moving and minor versions may break it. Maintained actively; issues are triaged as time allows.

## Why this instead of the `container` CLI

Without a library, a test that needs a database means `exec`-ing `container create`, `container start`, `container inspect`, parsing the inspect JSON for the container IP and published ports, polling until the port answers, and deleting the container in every failure path. This package is that sequence, plus a readiness strategy set and cleanup hooks, behind one call:

```go
c, err := applecontainer.Run(ctx, "postgres:16-alpine",
	applecontainer.WithExposedPorts("5432"),
	applecontainer.WithWaitStrategy(wait.ForSQL("5432", "pgx", dburl)),
)
```

If you already use `testcontainers-go` against Docker, the API and wait strategies are close enough that swapping the import is most of the port.

### When not to use it

- **Not on Linux, not on Intel Macs.** It shells out to Apple's `container` binary, which only exists on macOS 26+ on Apple Silicon.
- **It is not a Docker replacement.** There is no Docker API, no Compose, no socket. Tooling that speaks the Docker API (Testcontainers Cloud, `docker compose` fixtures, `dockerd`-based CI) stays on Docker and `testcontainers-go`.
- **Startup parallelism and file copies are slower than Docker** — see [Performance](#performance) for the numbers, including a 10x gap at 8 concurrent starts.

## Requirements

| | |
|:--|:--|
| Host | macOS 26+ on Apple Silicon |
| Go | 1.27 or newer |
| Runtime | Apple's [`container` CLI](https://github.com/apple/container) on `PATH`, with its service running |

`Run` fails with `XPC connection error: Connection invalid` if the runtime service is not up. Start it once:

```bash
container system start
container system status   # should report: status running
```

## Install

Inside an existing Go module (`go mod init` first if you have none):

```bash
go get github.com/lynicis/applecontainer-go
```

## Quickstart

Starts nginx, waits for HTTP, and prints the response status:

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c, err := applecontainer.Run(ctx, "nginx:alpine",
		applecontainer.WithExposedPorts("80"),
		applecontainer.WithWaitStrategy(wait.ForHTTP("/").WithPort("80")),
	)
	if err != nil {
		panic(err)
	}
	defer c.Terminate(ctx)

	endpoint, err := c.Endpoint(ctx, "80")
	if err != nil {
		panic(err)
	}

	resp, err := http.Get("http://" + endpoint)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Nginx status: %s\n", resp.Status)
}
```

Expected output on macOS 26 / Apple Silicon with the runtime up:

```
Nginx status: 200 OK
```

The first run also pulls `nginx:alpine`; expect a few seconds of that on a cold image cache.

## Networking

Two modes, chosen per container.

### Direct IP (default)

Ports are not published to the host. `c.Host(ctx)` returns the container's bridge IP and `c.Endpoint(ctx, "80")` returns `<container-ip>:80`, so the test dials the container directly and no host ports are allocated.

### Host-port mapping

Set `WithHostPortMapping(true)` to publish each exposed port to an ephemeral `127.0.0.1` port. `c.Host(ctx)` returns `localhost` and `c.Endpoint(ctx, "80")` returns `localhost:<allocated-port>`.

```go
c, err := applecontainer.Run(ctx, "nginx:alpine",
	applecontainer.WithExposedPorts("80"),
	applecontainer.WithHostPortMapping(true),
	applecontainer.WithWaitStrategy(wait.ForHTTP("/").WithPort("80")),
)
```

Use this when the code under test can only reach `localhost` — for example a browser driver or a client that ignores IP overrides.

## Wait strategies

`WithWaitStrategy` sets one strategy wrapped in a 60-second deadline. Wrap it yourself for a different budget:

```go
applecontainer.WithWaitStrategy(wait.ForAll(wait.ForLog("ready"), wait.ForHTTP("/health")).WithDeadline(2*time.Minute))
```

| Strategy | Constructor | Ready when |
|:--|:--|:--|
| Listening port | `wait.ForListeningPort(port)` | A TCP/UDP port accepts connections |
| HTTP | `wait.ForHTTP(path)` | The endpoint answers; see `WithStatusCodeMatcher`, `WithResponseMatcher`, `WithPort`, `WithTLS` |
| Log | `wait.ForLog(pattern)` | Container logs contain the pattern — `AsRegexp()`, `WithOccurrence(n)` |
| Exec | `wait.ForExec(cmd)` | The command exits 0 (or matches `WithExitCodeMatcher`/`WithResponseMatcher`) |
| SQL | `wait.ForSQL(port, driver, dburl)` | The query returns — `WithQuery(...)` |
| File | `wait.ForFile(path)` | The path exists in the container |
| Health | `wait.ForHealth()` | State is `running` with exit code 0 |
| All | `wait.ForAll(strats...)` | Every strategy passes, run sequentially |
| Any | `wait.ForAny(strats...)` | Any one strategy passes; the rest are cancelled |
| Exit | `wait.ForExit()` | The container exits (for one-shot containers) |

```go
// Postgres using the pgx driver
dburl := func(host string, port int) string {
	return fmt.Sprintf("postgres://postgres:postgres@%s:%d/postgres?sslmode=disable", host, port)
}
waitPostgres := wait.ForSQL("5432", "pgx", dburl).WithQuery("SELECT 1")

// An HTTP health endpoint returning 200
waitApp := wait.ForHTTP("/health").
	WithPort("8080").
	WithStatusCodeMatcher(func(code int) bool { return code == http.StatusOK })

// A log line appearing three times
waitLogs := wait.ForLog("server started").WithOccurrence(3)
```

## Networks and volumes

```go
nw, err := applecontainer.NewNetwork(ctx, applecontainer.NetworkRequest{
	Labels: map[string]string{"type": "integration"},
})
defer nw.Remove(ctx)

c1, err := applecontainer.Run(ctx, "nginx:alpine",
	applecontainer.WithNetwork([]string{"web-server"}, nw),
)
```

```go
vol, err := applecontainer.NewVolume(ctx, applecontainer.VolumeRequest{Size: "10G"})
defer vol.Remove(ctx)

c, err := applecontainer.Run(ctx, "postgres:16-alpine",
	applecontainer.WithVolumes(applecontainer.VolumeMount{
		Source: vol.Name(),
		Target: "/var/lib/postgresql/data",
	}),
)
```

`WithNewNetwork(ctx, aliases, NetworkRequest{...})` creates and attaches a network in one step and deletes it when `Run` fails.

## Common options

Request customizers are functional options; `Run` requires the image as its second argument unless you build one.

| Option | Purpose |
|:--|:--|
| `WithEnv(map[string]string)`, `WithCmd(...string)`, `WithEntrypoint(...string)` | Process configuration |
| `WithExposedPorts(...string)` | Ports the container exposes, resolved later via `Endpoint`/`MappedPort` — `"80"` or `"80/tcp"` |
| `WithHostPortMapping(bool)` | Publish exposed ports to ephemeral host ports |
| `WithNetworks(...string)`, `WithNetwork(aliases, *Network)`, `WithNetworkName(string)` | Network attachment |
| `WithVolumes(...VolumeMount)`, `WithTmpfs(map[string]string)`, `WithMounts(...Mount)` | Mounts |
| `WithCPUs(float64)`, `WithMemory(int64)`, `WithShmSize(int64)`, `WithUlimits(...Ulimit)` | Resource limits |
| `WithWaitStrategy(wait.Strategy)`, `WithLogWriters(...io.Writer)` | Readiness and log streaming |
| `WithFiles(...ContainerFile)`, `WithWorkingDir(string)`, `WithUser(string)`, `WithReadOnlyRootfs(bool)` | Filesystem and process identity |
| `WithAlwaysPull(bool)`, `WithPlatform(string)`, `WithArch(string)`, `WithOS(string)` | Image selection |
| `WithContainerfile(FromContainerfile)` | Build the image first (see below) |

The full exported surface is on [pkg.go.dev](https://pkg.go.dev/github.com/lynicis/applecontainer-go).

### Building from a Containerfile

`WithContainerfile` builds from a context directory and runs the result; the image argument to `Run` is then ignored. `FromContainerfile` takes `Context`, `File`, `BuildArgs`, `Tags`, `Target`, `Platform`, `NoCache`, `Pull`, `Secrets` and `KeepImage`.

```go
c, err := applecontainer.Run(ctx, "",
	applecontainer.WithContainerfile(applecontainer.FromContainerfile{
		Context: "./testdata/app",          // build context
		File:    "./testdata/app/Containerfile",
		Tags:    []string{"app:test"},
	}),
	applecontainer.WithWaitStrategy(wait.ForLog("listening")),
)
```

## Performance

Every benchmark runs the same operation against this library and `testcontainers-go` on Docker, via the `RunWithBoth` harness. Reproduce on your machine:

```bash
cd benchmarks && APPLECONTAINER_BENCHMARK=1 go test -bench=. -benchmem -benchtime=5x ./...
```

One local run, Go 1.27, darwin/arm64, Apple M5 — re-run before trusting the deltas:

| Operation | `applecontainer-go` | `testcontainers-go` | Faster |
|:--|--:|--:|:--:|
| Stop | 230ms | 187ms | Docker (1.2x) |
| Start (restart) | 36.6ms | 247ms | Apple (6.7x) |
| Terminate | 270ms | 206ms | Docker (1.3x) |
| Inspect | 252ms | 215ms | Docker (1.2x) |
| Exec (`echo hello`) | 297ms | 332ms | Apple (1.1x) |
| Copy To (1 KB) | 281ms | 224ms | Docker (1.3x) |
| Copy To (1 MB) | 273ms | 221ms | Docker (1.2x) |
| Copy From (1 KB) | 271ms | 209ms | Docker (1.3x) |
| Copy From (1 MB) | 253ms | 212ms | Docker (1.2x) |
| Logs (10k lines) | 5.47s | 10.22s | Apple (1.9x) |
| Wait: HTTP | 595ms | 336ms | Docker (1.8x) |
| Wait: SQL (pgx) | 1.62s | 1.05s | Docker (1.5x) |
| Wait: Exec | 648ms | 354ms | Docker (1.8x) |
| Wait: Health | 609ms | 427ms | Docker (1.4x) |
| Wait: Composite | 795ms | 414ms | Docker (1.9x) |
| Wait: Log (spam) | 581ms | 338ms | Docker (1.7x) |
| Parallel startup (2) | 1.10s | 280ms | Docker (3.9x) |
| Parallel startup (4) | 2.17s | 331ms | Docker (6.6x) |
| Parallel startup (8) | 4.40s | 434ms | Docker (10.1x) |
| TCP Latency (Redis) | 280ms | 367ms | Apple (1.3x) |

PostgreSQL driver microbenchmarks (`µs/op`), each driver set running Postgres under the respective runtime:

| Operation | Driver | `applecontainer-go` | `testcontainers-go` | Faster |
|:--|:--|--:|--:|:--:|
| Insert | pgx | 2156µs | 2306µs | Apple (1.1x) |
| | pq | 271µs | 186µs | Docker (1.5x) |
| | GORM | 1029µs | 781µs | Docker (1.3x) |
| Bulk insert (100 rows) | pgx | 457µs | 477µs | Apple (1.0x) |
| | pq | 493µs | 514µs | Apple (1.0x) |
| | GORM | 668µs | 749µs | Apple (1.1x) |
| Select by primary key | pgx | 379µs | 477µs | Apple (1.3x) |
| | pq | 242µs | 350µs | Apple (1.4x) |
| | GORM | 461µs | 526µs | Apple (1.1x) |
| Select with filter | pgx | 259µs | 649µs | Apple (2.5x) |
| | pq | 206µs | 421µs | Apple (2.0x) |
| | GORM | 218µs | 418µs | Apple (1.9x) |
| Update | pgx | 399µs | 461µs | Apple (1.2x) |
| | pq | 334µs | 668µs | Apple (2.0x) |
| | GORM | 427µs | 579µs | Apple (1.4x) |
| Delete | pgx | 195µs | 132µs | Docker (1.5x) |
| | pq | 178µs | 123µs | Docker (1.4x) |
| | GORM | 174µs | 147µs | Docker (1.2x) |

Reading the tables: restarts, bulk log retrieval and direct TCP access are the wins. Parallel startup is the loss — up to 10x at 8 concurrent containers — and copy operations pay for spawning `container cp` per transfer. In the driver benchmarks, applecontainer-go wins selects and updates; Docker wins single-row inserts (pq, GORM) and deletes.

## Development

```bash
make test            # hermetic, no container runtime needed
make test-examples   # integration; requires macOS + a running `container` service
make test-benchmark  # head-to-head benchmarks
make lint
```

`make test-examples` and `make test-benchmark` set the `APPLECONTAINER_INTEGRATION` / `APPLECONTAINER_BENCHMARK` environment variables themselves. `Terminate` — usually via `applecontainer.CleanupContainer(t, c)` — deletes the container, and `applecontainer.Prune` clears leftovers from crashed runs.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development setup and review checklist.

## License

MIT — see [LICENSE](LICENSE).
