# 🍎 applecontainer-go

[![Go Reference](https://pkg.go.dev/badge/github.com/lynicis/applecontainer-go.svg)](https://pkg.go.dev/github.com/lynicis/applecontainer-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/lynicis/applecontainer-go)](https://goreportcard.com/report/github.com/lynicis/applecontainer-go)
[![codecov](https://codecov.io/gh/lynicis/applecontainer-go/branch/main/graph/badge.svg)](https://codecov.io/gh/lynicis/applecontainer-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

`applecontainer-go` is a lightweight, `testcontainers-go`-style Go library designed to spin up Apple Container (`container` CLI) Linux containers as test dependencies on macOS.

Unlike Docker-based libraries, `applecontainer-go` integrates directly with the native Apple Silicon container virtualization engine, letting you boot up test dependencies like databases, web servers, and queues with near-zero overhead.

---

## Table of Contents

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

Performance comparison: **applecontainer-go** (Apple native `container` CLI) vs **testcontainers-go** (Docker Desktop).

**Hardware**: Apple M5 · macOS 26 · container CLI v1.0.0 · Docker Desktop v29.4.0
**Method**: `benchtime=1x`, single iteration per benchmark.

### Cold Start (3 images)

| Container | applecontainer-go | testcontainers-go | Δ |
| :--- | ---: | ---: | ---: |
| nginx:alpine | 1.84s | — | — |
| redis:alpine | 1.84s | — | — |
| postgres:alpine | 7.38s | — | — |

### Operation Latency

| Operation | applecontainer-go | testcontainers-go | Δ |
| :--- | ---: | ---: | ---: |
| Stop | 247ms | 260ms | **1.1x faster** |
| Start | 88ms | 248ms | **2.8x faster** |
| Terminate | 266ms | 283ms | **1.1x faster** |
| Inspect | 307ms | 295ms | ≈ same |
| Exec (echo) | 347ms | 425ms | **1.2x faster** |
| Copy To (1 KB) | 301ms | 297ms | ≈ same |
| Copy To (1 MB) | 290ms | 298ms | ≈ same |
| Copy From (1 KB) | 293ms | 280ms | ≈ same |
| Copy From (1 MB) | 307ms | 279ms | 1.1x slower |
| Logs (10k lines) | 5.55s | 10.24s | **1.8x faster** |

### Wait Strategies (Overhead)

Apple's native `container` CLI requires spawning new processes for `exec` and `inspect` loops, resulting in higher polling latency compared to Testcontainers' direct daemon socket connection.

| Strategy | applecontainer-go | testcontainers-go | Δ |
| :--- | ---: | ---: | ---: |
| HTTP | 747ms | 355ms | 2.1x slower |
| SQL | 1.74s | 518ms | 3.3x slower |
| Exec | 738ms | 321ms | 2.2x slower |
| Health | 711ms | 645ms | 1.1x slower |
| Composite (All) | 773ms | 433ms | 1.7x slower |
| Log (5k lines) | 756ms | 651ms | 1.1x slower |

### Internal Processing

| Operation | Latency | Memory |
| :--- | ---: | ---: |
| Parse Inspect JSON | 43µs | 81 allocs/op |
| Parse Image Inspect | 16µs | 9 allocs/op |

### Network Performance

| Test | applecontainer-go | testcontainers-go | Δ |
| :--- | ---: | ---: | ---: |
| TCP Latency (redis PING) | 414ms | 480ms | **1.2x faster** |
| HTTP Throughput (nginx) | N/A | 2.07s | — |

### Parallel Startup (nginx:alpine)

| Containers | applecontainer-go | testcontainers-go | Δ |
| ---: | ---: | ---: | ---: |
| 2 | 1.60s | 592ms | 2.7x slower |
| 4 | 6.22s | 966ms | 6.4x slower |
| 8 | 6.42s | 1.80s | 3.6x slower |

### Image Operations

| Test | applecontainer-go | testcontainers-go | Δ |
| :--- | ---: | ---: | ---: |
| Image Pull (postgres:alpine) | 9.97s | 2.56s | 3.9x slower |
| Image Build | N/A | 329ms | — |

**Key takeaways**:
- **Start/Stop/Terminate/Exec** are significantly faster with the native Apple runtime — up to 3x for Start.
- **Parallel scalability** is limited — the Apple CLI processes containers sequentially, while Docker runs them concurrently.
- **Image operations** (pull, build) are slower due to the Apple CLI's image management overhead.
- **HTTP throughput** from host is not supported in Apple's direct-IP networking model.

Run benchmarks locally:
```bash
make test-benchmark
```

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
