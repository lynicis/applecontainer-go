# Benchmark Plan: applecontainer-go vs testcontainers-go

**Date:** 2026-06-19
**Goal:** Produce reproducible benchmarks comparing `applecontainer-go` (Apple native `container` CLI) against `testcontainers-go` (Docker SDK) on macOS Apple Silicon. Results displayed in a README comparison table.

---

## 1. Benchmark Images

Three industry-standard test dependency images covering light / medium / heavy profiles:

| Image | Size | Wait Strategy | Profile |
|-------|------|---------------|---------|
| `nginx:alpine` | ~7 MB | `wait.ForHTTP("/")` | Light — web server |
| `redis:alpine` | ~12 MB | `wait.ForLog("* Ready*")` | Medium — in-memory data store |
| `postgres:alpine` | ~90 MB | `wait.ForSQL("5432", "pgx", ...)` | Heavy — database with SQL driver init |

---

## 2. Module Layout

Separate Go module (avoids adding testcontainers-go as a root dependency).

```
benchmarks/
├── go.mod                  # module github.com/lynicis/applecontainer-go/benchmarks
│                           #   go 1.26
│                           #   require github.com/testcontainers/testcontainers-go v0.42.0
│                           #   replace github.com/lynicis/applecontainer-go => ../
├── go.sum
├── compare.go              # Runtime enum, RunWithBoth, Skip helpers
├── startup_test.go         # Container cold/warm start latency (3 images × 2 runtimes)
├── operations_test.go      # Per-operation latency (create, start, stop, inspect, exec, copy)
├── parallel_test.go        # Concurrent container scalability (2, 4, 8)
├── network_test.go         # HTTP throughput, TCP round-trip
└── image_test.go           # Image pull time, build from Containerfile
```

Build constraint: `//go:build benchmark` on every `_test.go` file + `APPLECONTAINER_BENCHMARK=1` env guard.

---

## 3. Benchmark Dimensions & Scenarios

### 3a. Startup Time (`startup_test.go`)

Each benchmark:
1. Pull image (outside timer)
2. `b.ResetTimer()`
3. `Run(ctx, img, opts...)` + wait strategy
4. `b.StopTimer()` once container reports ready
5. Terminate container

```
BenchmarkStartup/applecontainer/nginx-alpine
BenchmarkStartup/testcontainers-go/nginx-alpine
BenchmarkStartup/applecontainer/redis-alpine
BenchmarkStartup/testcontainers-go/redis-alpine
BenchmarkStartup/applecontainer/postgres-alpine
BenchmarkStartup/testcontainers-go/postgres-alpine
```

### 3b. Operation Latency (`operations_test.go`)

Measure individual lifecycle calls. Container is pre-started; `b.ResetTimer()` before each operation.

| Benchmark | applecontainer | testcontainers |
|-----------|---------------|----------------|
| `CreateContainer` | `container create` | `Client.ContainerCreate` |
| `StartContainer` | `container start` | `Client.ContainerStart` |
| `StopContainer` | `container stop` | `Client.ContainerStop` |
| `TerminateContainer` | `container stop` + `delete` | `Client.ContainerRemove` |
| `InspectContainer` | `container inspect` → parse JSON | `Client.ContainerInspect` |
| `ExecContainer` | `container exec echo hello` | `Client.ContainerExecCreate` + `Attach` |
| `CopyFile/1KB` | `container cp` 1 KB file | `Client.CopyToContainer` |
| `CopyFile/1MB` | `container cp` 1 MB file | `Client.CopyToContainer` |

Metrics: `ns/op`, `B/op`, `allocs/op` (via `-benchmem`).

### 3c. Parallel Scalability (`parallel_test.go`)

Start N containers concurrently (all nginx:alpine, same image cached), measure total wall time.

```
BenchmarkParallel/applecontainer/2
BenchmarkParallel/applecontainer/4
BenchmarkParallel/applecontainer/8
BenchmarkParallel/testcontainers-go/2
BenchmarkParallel/testcontainers-go/4
BenchmarkParallel/testcontainers-go/8
```

### 3d. Network Performance (`network_test.go`)

| Benchmark | What it measures |
|-----------|-----------------|
| `HTTPThroughput` | Concurrent HTTP GET throughput (nginx, 100 requests) |
| `TCPLatency` | Redis PING round-trip time (exec inside container) |

### 3e. Image Operations (`image_test.go`)

| Benchmark | What |
|-----------|------|
| `ImagePull/nginx-alpine` | Time to pull cached image (second pull) |
| `ImageBuild` | Build a trivial Containerfile (`echo hello > /usr/share/nginx/html/index.html`) |

---

## 4. Shared Infrastructure (`compare.go`)

```go
type Runtime int
const (
    AppleContainer Runtime = iota
    TestcontainersGo
)

// RunWithBoth runs the benchmark function for both runtimes in sub-benchmarks.
func RunWithBoth(b *testing.B, fn func(b *testing.B, rt Runtime))

// SkipIfDockerNotHealthy skips if Docker is unavailable.
func SkipIfDockerNotHealthy(t testing.TB)

// ImageRef resolves image name (handles HubImagePrefix for applecontainer).
func ImageRef(name string) string

// ContainerName generates a unique container name per benchmark run.
func ContainerName(b *testing.B) string
```

---

## 5. Running

```bash
# Ensure both runtimes are warm
container system start
open -a Docker               # or ensure Docker Desktop is running

APPLECONTAINER_BENCHMARK=1 go test -tags benchmark -bench=. -benchmem \
  -benchtime=3x \
  ./benchmarks/ 2>&1 | tee bench-raw.txt

# Statistical analysis (optional)
go install golang.org/x/perf/cmd/benchstat@latest
benchstat bench-raw.txt
```

No CI integration — manual run on demand.

---

## 6. README Section (to fill in after running)

> **Note:** Filled in after first benchmark run on target hardware.

```markdown
## Benchmarks

Measured on [hardware, e.g. MacBook Pro M4 Pro, macOS 26.0, 24 GB RAM].
Apple Container runtime v1.0.0, Docker Desktop vX.Y.Z.

### Startup Time (lower is better)

| Container | applecontainer-go | testcontainers-go (Docker) | Speedup |
|-----------|------------------:|--------------------------:|--------:|
| nginx:alpine | — | — | — |
| redis:alpine | — | — | — |
| postgres:alpine | — | — | — |

### Operation Latency

| Operation | applecontainer-go | testcontainers-go |
|-----------|------------------:|------------------:|
| Create | — | — |
| Start | — | — |
| Stop | — | — |
| Terminate | — | — |
| Inspect | — | — |
| Exec | — | — |
| Copy 1 KB | — | — |
| Copy 1 MB | — | — |

### Parallel Startup (nginx:alpine)

| Containers | applecontainer-go | testcontainers-go |
|-----------:|------------------:|------------------:|
| 2 | — | — |
| 4 | — | — |
| 8 | — | — |
```

---

## 7. Implementation Order

1. `benchmarks/go.mod` + `go.sum` — module init with replace directive
2. `compare.go` — shared helpers, runtime enum, skip guards
3. `startup_test.go` — headline benchmark (all 6 combinations)
4. `operations_test.go` — per-operation latency breakdown
5. `parallel_test.go` — concurrency/scalability
6. `network_test.go` — throughput measurements
7. `image_test.go` — pull and build latency
8. Execute full suite, collect numbers, fill README table
