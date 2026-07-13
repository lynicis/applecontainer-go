# 🍎 applecontainer-go

[![Go Reference](https://pkg.go.dev/badge/github.com/lynicis/applecontainer-go.svg)](https://pkg.go.dev/github.com/lynicis/applecontainer-go)
[![codecov](https://codecov.io/gh/lynicis/applecontainer-go/branch/main/graph/badge.svg)](https://codecov.io/gh/lynicis/applecontainer-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

`applecontainer-go` is a lightweight, `testcontainers-go`-style Go library designed to spin up Apple Container (`container` CLI) Linux containers as test dependencies on macOS.

Unlike Docker-based libraries, `applecontainer-go` integrates directly with the native Apple Silicon container virtualization engine, letting you boot up test dependencies like databases, web servers, and queues with near-zero overhead.

---

## Table of Contents

- [Documentation](#documentation)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Install](#install)
- [Quickstart](#quickstart)
- [Networking Models](#networking-models)
  - [1. Direct IP Mode (Default)](#1-direct-ip-mode-default)
  - [2. Host-Port Mapping Mode (Opt-in)](#2-host-port-mapping-mode-opt-in)
- [Wait Strategies](#wait-strategies)
- [Customizer Options](#customizer-options)
- [Networks and Volumes](#networks-and-volumes)
  - [Networks](#networks)
  - [Volumes](#volumes)
- [Benchmarks](#benchmarks)
- [Testing and Verification](#testing-and-verification)
- [Contributing](#contributing)
- [License](#license)

---

## Documentation

**[Interactive documentation](https://gistcdn.githack.com/lynicis/1b74df1662a9c68ebfd7cc76c5077578/raw/639ef50a24db1e71ea89d2b2608ea226f48fd969/index.html)**

---

## Features

- 🏎️ **Fast Boot times**: Harness macOS native container technology for minimal latency.
- ⚓ **Familiar API**: Modelled closely after `testcontainers-go` to ensure a minimal learning curve.
- 🌐 **Double Networking Modes**: Choose between Bridged Direct IP and Host-Port mapping models.
- ⏱️ **Robust Wait Strategies**: Built-in support for HTTP, listening ports, logs, SQL databases, executions, and file existence.
- 🏗️ **Build from Context**: Support for building images on the fly via `Containerfile` / `Dockerfile`.

---

## Prerequisites

- **Host OS**: macOS 26+ (Apple native container orchestration environment)
- **Architecture**: Apple Silicon (M1, M2, M3, M4, or newer)
- **Dependency**: The Apple native `container` CLI must be installed and active (e.g. `/opt/homebrew/bin/container` or `/usr/local/bin/container`).

---

## Install

```bash
go get -u github.com/lynicis/applecontainer-go
```

---

## Quickstart

Spin up an Nginx container with an HTTP wait strategy and get the endpoint:

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

	// Spin up Nginx in direct container IP mode
	c, err := applecontainer.Run(ctx, "nginx:alpine",
		applecontainer.WithExposedPorts("80"),
		applecontainer.WithWaitStrategy(wait.ForHTTP("/").WithPort("80")),
	)
	if err != nil {
		panic(err)
	}
	defer c.Terminate(ctx)

	// Fetch endpoint (resolves to direct container IP:80 by default)
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

---

## Networking Models

`applecontainer-go` supports two distinct networking architectures:

### 1. Direct IP Mode (Default)
In this mode, container virtual interfaces map directly onto the macOS host bridge.
- You can communicate directly with the container's private bridge IP.
- `c.Host(ctx)` resolves to the container's IP address.
- `c.Endpoint(ctx, port)` matches the container's IP and port directly.
- **Limitation**: No host ports are consumed, making it ideal for running many concurrent integration tests.

### 2. Host-Port Mapping Mode (Opt-in)
To bind container ports to ephemeral host ports on the localhost interface, enable this option using `WithHostPortMapping(true)`.
- Ephemeral host ports are automatically allocated and bound to the host network interface.
- `c.Host(ctx)` resolves to `"localhost"`.
- `c.Endpoint(ctx, port)` resolves to `localhost:<random_host_port>`.

```go
c, err := applecontainer.Run(ctx, "nginx:alpine",
	applecontainer.WithExposedPorts("80"),
	applecontainer.WithHostPortMapping(true),
	applecontainer.WithWaitStrategy(wait.ForHTTP("/").WithPort("80")),
)
```

---

## Wait Strategies

A container isn't always fully initialized as soon as it starts. Wait strategies allow you to block execution until the dependency is ready.

| Strategy | Constructor | Description |
| :--- | :--- | :--- |
| **Listening Port** | `wait.ForListeningPort(port)` | Checks if a TCP/UDP port is open and listening. |
| **HTTP** | `wait.ForHTTP(path)` | Performs HTTP requests and validates status codes/response body matchers. |
| **Log Stream** | `wait.ForLog(pattern)` | Scans stdout/stderr logs for a substring or regexp pattern. |
| **Exec Command** | `wait.ForExec(cmd)` | Executes a command inside the container repeatedly until exit code or outputs match. |
| **SQL Database** | `wait.ForSQL(port, driver, dburl)` | Verifies database availability using standard Go `database/sql` driver and query. |
| **File Check** | `wait.ForFile(path)` | Verifies file existence inside the container filesystem. |
| **Container Health** | `wait.ForHealth()` | Waits for the container state to be `"running"` with exit code `0`. |
| **All (Composite)** | `wait.ForAll(strats...)` | Runs all provided wait strategies sequentially. |
| **Any (Composite)** | `wait.ForAny(strats...)` | Runs all provided wait strategies concurrently, succeeding when any one does. |

### Wait Strategy Examples

```go
// Wait for Postgres using SQL driver pgx
dburl := func(host string, port int) string {
	return fmt.Sprintf("postgres://postgres:postgres@%s:%d/postgres?sslmode=disable", host, port)
}
waitPostgres := wait.ForSQL("5432", "pgx", dburl).WithQuery("SELECT 1")

// Wait for a application server healthcheck endpoint
waitApp := wait.ForHTTP("/health").
	WithPort("8080").
	WithStatusCodeMatcher(func(code int) bool { return code == http.StatusOK })

// Wait for a log occurrence 3 times
waitLogs := wait.ForLog("server started").WithOccurrence(3)
```

---

## Customizer Options

Customize the container configuration using functional options:

| Customizer Option | Type / Arguments | Purpose |
| :--- | :--- | :--- |
| `WithImage(img)` | `string` | Defines the container image. |
| `WithExposedPorts(ports...)` | `...string` | Ports exposed from the container (e.g. `"80/tcp"`). |
| `WithHostPortMapping(bool)` | `bool` | Enables mapping exposed ports to host localhost ephemeral ports. |
| `WithEnv(map)` | `map[string]string` | Key-value environment variables. |
| `WithCmd(cmd...)` | `...string` | Command parameters passed to the container. |
| `WithEntrypoint(entry...)` | `...string` | Container entrypoint command. |
| `WithNetworks(names...)` | `...string` | Custom network names to attach the container to. |
| `WithVolumes(vols...)` | `...VolumeMount` | Directory volumes to mount. |
| `WithTmpfs(map)` | `map[string]string` | Mount tmpfs directories. |
| `WithCPUs(cpus)` | `float64` | CPU allocation limits. |
| `WithMemory(bytes)` | `int64` | Memory allocation limits. |
| `WithWaitStrategy(strat)` | `wait.Strategy` | Configures the default wait strategy with a default 60s timeout. |
| `WithContainerfile(cf)` | `FromContainerfile` | Build image from a context directory using Containerfile. |
| `WithLogWriters(w...)` | `...io.Writer` | Register `io.Writer`s to receive streaming container stdout/stderr logs. |

---

## Networks and Volumes

### Networks
Create and attach containers to custom networks to facilitate inter-container communication:

```go
nw, err := applecontainer.NewNetwork(ctx,
	applecontainer.WithNetworkLabels(map[string]string{"type": "integration"}),
)
defer nw.Remove(ctx)

c1, err := applecontainer.Run(ctx, "nginx:alpine",
	applecontainer.WithNetwork([]string{"web-server"}, nw),
)
```

### Volumes
Easily create volumes and mount them to containers to persist data across container lifecycles:

```go
vol, err := applecontainer.NewVolume(ctx,
	applecontainer.WithVolumeSize("10G"),
)
defer vol.Remove(ctx)

c, err := applecontainer.Run(ctx, "postgres:alpine",
	applecontainer.WithVolumes(applecontainer.VolumeMount{
		Source: vol.Name(),
		Target: "/var/lib/postgresql/data",
	}),
)
```

---

## Benchmarks

Head-to-head `applecontainer-go` vs `testcontainers-go` (Docker). Every benchmark runs the same operation on both runtimes via the `RunWithBoth` harness.

```bash
cd benchmarks && APPLECONTAINER_BENCHMARK=1 go test -bench=. -benchmem -benchtime=5x ./...
```

| Operation | `applecontainer-go` | `testcontainers-go` | Winner |
| :--- | ---: | ---: | :---: |
| Stop | 230ms | 187ms | **🐳 Docker** (1.2x faster) |
| Start (restart) | 36.6ms | 247ms | **🍎 Apple** (6.7x faster) |
| Terminate | 270ms | 206ms | **🐳 Docker** (1.3x faster) |
| Inspect | 252ms | 215ms | **🐳 Docker** (1.2x faster) |
| Exec (`echo hello`) | 297ms | 332ms | **🍎 Apple** (1.1x faster) |
| Copy To (1 KB) | 281ms | 224ms | **🐳 Docker** (1.3x faster) |
| Copy To (1 MB) | 273ms | 221ms | **🐳 Docker** (1.2x faster) |
| Copy From (1 KB) | 271ms | 209ms | **🐳 Docker** (1.3x faster) |
| Copy From (1 MB) | 253ms | 212ms | **🐳 Docker** (1.2x faster) |
| Logs (10k lines) | 5.47s | 10.22s | **🍎 Apple** (1.9x faster) |
| Wait: HTTP | 595ms | 336ms | **🐳 Docker** (1.8x faster) |
| Wait: SQL (pgx) | 1.62s | 1.05s | **🐳 Docker** (1.5x faster) |
| Wait: Exec | 648ms | 354ms | **🐳 Docker** (1.8x faster) |
| Wait: Health | 609ms | 427ms | **🐳 Docker** (1.4x faster) |
| Wait: Composite | 795ms | 414ms | **🐳 Docker** (1.9x faster) |
| Wait: Log (spam) | 581ms | 338ms | **🐳 Docker** (1.7x faster) |
| Parallel startup (2) | 1.10s | 280ms | **🐳 Docker** (3.9x faster) |
| Parallel startup (4) | 2.17s | 331ms | **🐳 Docker** (6.6x faster) |
| Parallel startup (8) | 4.40s | 434ms | **🐳 Docker** (10.1x faster) |
| TCP Latency (Redis) | 280ms | 367ms | **🍎 Apple** (1.3x faster) |

**PostgreSQL driver microbenchmarks** (`µs/op`) — each driver set hosts Postgres under the respective container runtime:

| Operation | Driver | `applecontainer-go` | `testcontainers-go` | Winner |
| :--- | :--- | ---: | ---: | :---: |
| Insert | pgx | 2156µs | 2306µs | 🍎 Apple (1.1x) |
| | pq | 271µs | 186µs | 🐳 Docker (1.5x) |
| | GORM | 1029µs | 781µs | 🐳 Docker (1.3x) |
| Bulk insert (100 rows) | pgx | 457µs | 477µs | 🍎 Apple (1.0x) |
| | pq | 493µs | 514µs | 🍎 Apple (1.0x) |
| | GORM | 668µs | 749µs | 🍎 Apple (1.1x) |
| Select by primary key | pgx | 379µs | 477µs | 🍎 Apple (1.3x) |
| | pq | 242µs | 350µs | 🍎 Apple (1.4x) |
| | GORM | 461µs | 526µs | 🍎 Apple (1.1x) |
| Select with filter | pgx | 259µs | 649µs | 🍎 Apple (2.5x) |
| | pq | 206µs | 421µs | 🍎 Apple (2.0x) |
| | GORM | 218µs | 418µs | 🍎 Apple (1.9x) |
| Update | pgx | 399µs | 461µs | 🍎 Apple (1.2x) |
| | pq | 334µs | 668µs | 🍎 Apple (2.0x) |
| | GORM | 427µs | 579µs | 🍎 Apple (1.4x) |
| Delete | pgx | 195µs | 132µs | 🐳 Docker (1.5x) |
| | pq | 178µs | 123µs | 🐳 Docker (1.4x) |
| | GORM | 174µs | 147µs | 🐳 Docker (1.2x) |

*(Specs: Go 1.26.5 · darwin/arm64 · Apple M5; single-iteration benchmarks. Run the suite yourself to get multi-sample averages.)*

**Takeaway:** `applecontainer-go` is especially strong at restarts, bulk log retrieval (1.9x faster than Docker for 10k lines), and direct container TCP access. Docker remains faster for highly parallel startup, exec-heavy waits, and copy operations — parallel startup is Docker's strongest advantage (up to 10x at 8 concurrent containers). The copy gap is roughly 1.2-1.3x and is dominated by spawning `container cp` for each transfer. In the PostgreSQL driver benchmarks, applecontainer-go wins most operations across all three drivers — especially selects (up to 2.5x faster for filtered queries) and updates (up to 2.0x). Docker wins single-row inserts (pq, GORM) and deletes. Run the suite yourself to verify on your hardware.

---

## Testing and Verification

### Unit Tests
Unit tests use mocking and system CLI fakes and do not require the actual Apple container runtime environment.
```bash
make test
```

### Integration Tests
Integration tests run real container scenarios and require macOS with a running `container` CLI daemon.
```bash
make test-examples
```

---

## Contributing

We welcome issues, bug reports, and pull requests to improve `applecontainer-go`.

Please read our [Contributing Guide](CONTRIBUTING.md) for details on how to set up the local development environment, run tests, and follow code quality guidelines.

Briefly:
1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'Add amazing feature'`).
4. Push to the branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request.

---

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for more information.
