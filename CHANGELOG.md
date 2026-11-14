# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-11-14

### Added
- Added a `make fmt` target backed by golangci-lint formatting
- Added hermetic Go-layer microbenchmarks plus `make bench-hermetic`, sampled `make test-benchmark` runs and `make benchstat` comparison targets
- Added a CI benchmark smoke job that runs the hermetic benchmarks and compiles the runtime benchmark module

### Changed
- Benchmarks now measure the same operation on both runtimes: startup is a shared nginx/redis/postgres table, HTTP throughput and direct TCP round trips run against both, SQL wait uses `ForSQL` on both sides, and build benchmarks are cache-free on both
- Benchmark teardown is excluded from timers, failures fail the benchmark, and the harness pulls images once per process
- The README performance section reports `Exec (redis-cli ping)` and `Wait: running state (proxy)` under their measured names
- Adopted the Go 1.27 toolchain and pinned golangci-lint and gosec as Go tool dependencies
- `NetworkRequest` no longer takes `Driver` or `EnableIPv6`; `Config.Debug` is gone (the CLI reads `CONTAINER_DEBUG` itself)
- Containers are no longer created with `--rm`; `Terminate`/`CleanupContainer` remove them and `Prune` clears leftovers
- `Run` now terminates and deletes the container (running registered cleanups) when any step after create fails, returning a nil container instead of a half-started one
- Rewrote the README around adopting the library: status, tradeoffs versus the raw `container` CLI, when not to use it, and the `container system start` prerequisite
- Removed the dead interactive-docs link, the hand-written table of contents, and duplicate contributing steps from the README

### Removed
- Removed assistant-specific configuration files and bundled gitnexus skill documentation
- Removed `NetworkRequest.Driver`, `NetworkRequest.EnableIPv6`, `Config.Debug`, `MountTypeTmpfs` and unused logger fields

### Fixed
- Create flags now cover DNS, DNS domain, DNS search, `--no-dns`, ulimits and network aliases
- Ulimits with a zero soft limit are sent as `name=hard`
- `NewNetwork` no longer sends unsupported `--driver`/`--ipv6` flags
- `NewVolume` uses `-s` for size
- `WithAlwaysPull` actually pulls the image before create (skipped for Containerfile builds, which build the image locally)
- A failed version check is retried on the next `Run` instead of being cached forever
- `wait.ForExit` only treats not-found inspect errors as exit; transient errors keep polling
- `wait.ForLog` no longer leaks its scanner goroutine
- Logs include stderr for both followed and one-shot reads; error messages no longer echo full CLI arguments (secrets)
- `WithFiles` copies host files directly and treats mode `0` as `0644`
- `WithEntrypoint` keeps the entrypoint arguments after the program
- `HostPorts` accepts both `"80"` and `"80/tcp"` keys
- `wait.ForHTTP` preserves query strings in the path

## [0.2.0] - 2026-07-07

### Added
- Comprehensive benchmark suite comparing `applecontainer-go` vs `testcontainers-go`, covering driver operations, wait strategies, and JSON parsing.
- Issue and pull request templates to standardize community contributions.
- GitNexus documentation and agent skill definitions for code intelligence integration.

### Changed
- **Major refactor:** Purged over-engineered lifecycle hooks and wait abstractions in favor of simpler, more direct patterns.
- Replaced `github.com/goccy/go-json` with standard library `encoding/json` to reduce dependencies.
- Replaced custom log infrastructure with a thin `log/slog` wrapper.
- Isolated the `examples` folder into its own Go module.
- Integrated Codecov into the GitHub Actions CI pipeline.
- Standardized testdata formatting and added a `Makefile` for local workflows.
- Updated GitHub Actions versions and coverage badge to track `main` branch.
- Updated test-coverage output path and removed atomic mode.

### Removed
- Removed `ForExposedPort` and `ForMappedPort` port aliases.
- Removed empty `PullOption`, `TerminateOption`, dead `SessionID()` and `Close()`.
- Removed no-op public functions: `WithEnvFile`, `WithLogger`, `WithAdditionalLifecycleHooks`.
- Removed unused `StatsEntry`, `ListEntry`, `parseList`, and `parseStats`.

### Fixed
- Resolved gosec security warnings across the codebase and updated CI workflows.
- Fixed `Makefile` benchmark target and Docker skip conditions.

### Security
- Created `SECURITY.md` to define security policy and vulnerability reporting guidelines.

### Documentation
- Added `CODE_OF_CONDUCT.md` to establish community guidelines.
- Added comprehensive contribution guidelines in `CONTRIBUTING.md`.
- Created `CHANGELOG.md` to track release history.
- Updated architecture documentation and cleaned up outdated plans.

## [0.1.0] - 2026-06-17

This is the initial release of `applecontainer-go`, a lightweight `testcontainers-go`-style Go library designed to run Linux containers on macOS via Apple's native container orchestration engine.

### Added
- Native Apple virtualization integration for container lifecycles (`start`, `stop`, `kill`, `delete`, `copy`, and `exec`).
- Flexible container customizer options (`WithExposedPorts`, `WithHostPortMapping`, `WithEnv`, `WithCPUs`, `WithMemory`, etc.).
- Complete set of Wait Strategies:
  - **Listening Port**: Checks if a port is open.
  - **HTTP**: Performs HTTP requests and validates status codes/body.
  - **Log Stream**: Scans stdout/stderr logs for substring or regex.
  - **Exec Command**: Executes a command inside the container repeatedly.
  - **SQL Database**: Verifies database availability using Go `database/sql` driver.
  - **File Check**: Verifies file existence inside the container.
  - **Container Health**: Wait for container state to be running.
- Volume and network creation and management APIs (`NewVolume`, `NewNetwork`).
- Interactive custom callback injection via `ContainerLifecycleHooks`.
- Custom `LogConsumer` interfacing to consume container log streams.
- Comprehensive integration and unit test suite.
- Comprehensive README documentation.
