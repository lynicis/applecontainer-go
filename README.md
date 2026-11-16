# applecontainer-go (experimental)

A Go library that runs Linux containers as test dependencies through Apple's native `container` CLI, with a `testcontainers-go`-shaped API.

[![Go Reference](https://pkg.go.dev/badge/github.com/lynicis/applecontainer-go.svg)](https://pkg.go.dev/github.com/lynicis/applecontainer-go)
[![codecov](https://codecov.io/gh/lynicis/applecontainer-go/branch/main/graph/badge.svg)](https://codecov.io/gh/lynicis/applecontainer-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**Status:** experimental, pre-1.0 (latest release `v0.4.0`). The API is still moving and minor versions may break it. Maintained actively; issues are triaged as time allows.

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
- **Startup parallelism and file copies are slower than Docker** — see [Performance](#performance) for the numbers, including a 7.6x gap at 8 concurrent starts.

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

Every benchmark runs the same operation against this library and `testcontainers-go` on Docker, via the `RunWithBoth` harness. Figures below are medians from five sampled runs. Reproduce on your machine:

```bash
make test-benchmark
cd benchmarks && go tool benchstat bench.txt
```

Use `go tool benchstat old.txt new.txt` (or `make benchstat BENCH_OLD=old.txt BENCH_NEW=new.txt`) to compare before/after runs.

Median of five sampled runs per operation, one local run, Go 1.27, darwin/arm64, Apple M5 — re-run before trusting the deltas:

| Operation | `applecontainer-go` | `testcontainers-go` | Faster |
|:--|--:|--:|:--:|
| Stop | 124ms | 144ms | Apple (1.2x) |
| Start (restart) | 390ms | 59.5ms | Docker (6.6x) |
| Terminate | 154ms | 151ms | Even |
| Inspect | 13.7ms | 466µs | Docker (29x) |
| Exec (`echo hello`) | 56.2ms | 115ms | Apple (2.0x) |
| Copy To (1 KB) | 22.8ms | 1.27ms | Docker (18x) |
| Copy To (1 MB) | 26.9ms | 5.74ms | Docker (4.7x) |
| Copy From (1 KB) | 18.6ms | 875µs | Docker (21x) |
| Copy From (1 MB) | 21.6ms | 2.35ms | Docker (9.2x) |
| Logs (10k lines) | 12.9ms | 16.1ms | Apple (1.2x) |
| HTTP throughput (nginx) | 12.4 MB/s | 11.1 MB/s | Apple (1.1x) |
| Wait: HTTP | 479ms | 210ms | Docker (2.3x) |
| Wait: SQL (pgx) | 1.09s | 1.17s | Apple (1.1x) |
| Wait: Exec | 533ms | 276ms | Docker (1.9x) |
| Wait: running state (proxy) | 478ms | 336ms | Docker (1.4x) |
| Wait: Composite | 495ms | 337ms | Docker (1.5x) |
| Wait: Log (spam) | 505ms | 261ms | Docker (1.9x) |
| Parallel startup (2) | 883ms | 225ms | Docker (3.9x) |
| Parallel startup (4) | 1.76s | 289ms | Docker (6.1x) |
| Parallel startup (8) | 3.57s | 466ms | Docker (7.6x) |
| Exec (redis-cli ping) | 61.3ms | 114ms | Apple (1.9x) |
| TCP round trip (Redis) | 47.3µs | 53.4µs | Apple (1.1x) |

Startup (create + start + ready) for the same wait strategy on both runtimes:

| Image | `applecontainer-go` | `testcontainers-go` | Faster |
|:--|--:|--:|:--:|
| nginx | 490ms | 209ms | Docker (2.3x) |
| redis | 524ms | 207ms | Docker (2.5x) |
| postgres | 854ms | 966ms | Apple (1.1x) |

PostgreSQL driver microbenchmarks (`µs/op`), each driver set running Postgres under the respective runtime:

| Operation | Driver | `applecontainer-go` | `testcontainers-go` | Faster |
|:--|:--|--:|--:|:--:|
| Insert | pgx | 123µs | 63.9µs | Docker (1.9x) |
| | pq | 198µs | 121µs | Docker (1.6x) |
| | GORM | 250µs | 190µs | Docker (1.3x) |
| Bulk insert (100 rows) | pgx | 300µs | 209µs | Docker (1.4x) |
| | pq | 399µs | 308µs | Docker (1.3x) |
| | GORM | 474µs | 359µs | Docker (1.3x) |
| Select by primary key | pgx | 65.0µs | 63.6µs | Even |
| | pq | 141µs | 123µs | Docker (1.1x) |
| | GORM | 67.9µs | 68.0µs | Even |
| Select with filter | pgx | 69.1µs | 71.9µs | Even |
| | pq | 128µs | 131µs | Even |
| | GORM | 83.6µs | 88.4µs | Apple (1.1x) |
| Update | pgx | 140µs | 71.5µs | Docker (2.0x) |
| | pq | 188µs | 137µs | Docker (1.4x) |
| | GORM | 240µs | 227µs | Docker (1.1x) |
| Delete | pgx | 305µs | 73.2µs | Docker (4.2x) |
| | pq | 193µs | 123µs | Docker (1.6x) |
| | GORM | 220µs | 180µs | Docker (1.2x) |

Reading the tables: `Exec`, log retrieval and direct TCP round trips are the wins (or near parity), and Postgres startup and the SQL wait strategy are close. Docker wins parallel startup (up to 7.6x at 8 concurrent containers), inspect, copies — which spawn a `container cp` per transfer — and single-row driver writes and deletes.

The Go layer also has hermetic microbenchmarks (`make bench-hermetic`) for option-to-CLI argument building, inspect parsing, the `exec` process floor, and the HTTP/log wait strategies.

## Development

```bash
make test            # hermetic, no container runtime needed
make test-examples   # integration; requires macOS + a running `container` service
make test-benchmark  # head-to-head benchmarks
make bench-hermetic  # Go-layer microbenchmarks; no container runtime needed
make lint
```

`make test-examples` and `make test-benchmark` set the `APPLECONTAINER_INTEGRATION` / `APPLECONTAINER_BENCHMARK` environment variables themselves. `Terminate` — usually via `applecontainer.CleanupContainer(t, c)` — deletes the container, and `applecontainer.Prune` clears leftovers from crashed runs.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development setup and review checklist.

## License

MIT — see [LICENSE](LICENSE).
