# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-11-14

### Added
- `make fmt` target backed by golangci-lint formatting
- Hermetic Go-layer microbenchmarks, plus `make bench-hermetic`, sampled `make test-benchmark` runs and `make benchstat` comparisons
- CI benchmark smoke job that runs the hermetic benchmarks and compiles the runtime benchmark module

### Changed
- Every benchmark now measures the same operation on both runtimes: startup uses a shared nginx/redis/postgres table, HTTP throughput and direct TCP round trips run against both, SQL wait uses `ForSQL` on both sides, and image builds are cache-free on both
- Benchmark teardown is excluded from timers, failed pulls fail the benchmark, and each image is pulled once per process
- The README performance section reports `Exec (redis-cli ping)` and `Wait: running state (proxy)` under their measured names
- Adopted the Go 1.27 toolchain and pinned golangci-lint and gosec as Go tool dependencies
- Containers are no longer created with `--rm`; `Terminate`/`CleanupContainer` remove them and `Prune` clears leftovers
- `Run` now terminates and deletes the container (running registered cleanups) when any step after create fails, returning a nil container instead of a half-started one
- Rewrote the README around adopting the library: status, tradeoffs versus the raw `container` CLI, when not to use it, and the `container system start` prerequisite; rewrote AGENTS.md as the developer guide
- Removed the dead interactive-docs link, the hand-written table of contents, and duplicate contributing steps from the README

### Removed
- Assistant-specific configuration files (`CLAUDE.md`, `opencode.json`) and the bundled gitnexus skill documentation
- `NetworkRequest.Driver`, `NetworkRequest.EnableIPv6`, `Config.Debug` (the CLI reads `CONTAINER_DEBUG` itself), `MountTypeTmpfs` and unused logger fields

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

## [0.3.0] - 2026-07-13

### Added
- `Provider` exported struct, replacing the `ContainerProvider` interface and the unexported `cliProvider`
- `wait.StrategyTarget.State()` returns status and exit code from one inspect; the health, HTTP, port and SQL strategies use it instead of re-inspecting
- Driver benchmarks running pgx, pq and GORM under both `applecontainer-go` and `testcontainers-go` (36 sub-benchmarks)
- Interactive documentation link in the README

### Changed
- `Container`, `Network` and `Volume` are concrete exported types instead of interfaces; `Run` returns `*Container`, `CleanupContainer` takes `*Container`, and `NewNetwork`/`NewVolume`/`WithNetwork` use `*Network`/`*Volume`
- `Config` lost `DefaultNetwork`, `DefaultPlatform`, `HubImagePrefix` and `PullTimeout`
- `SessionID` uses 16 random bytes instead of a pid/timestamp hash
- Wait targets cache the container host and mapped ports; the copy path skips a full read when the host file mode already matches
- Benchmark timers exclude setup, failed pulls fail the benchmark, and allocations are reported
- README benchmark tables refreshed (Apple M5, Go 1.26.5) with a head-to-head comparison section

### Removed
- `ImageInspect` type
- `WithWaitStrategyAndDeadline`, `WithAdditionalWaitStrategy`, `WithCLIArgsModifier` and `CLIArgsModifier`

### Fixed
- `waitTarget.Logs` tails with follow, so log-based waits see output written after the read starts
- `container image inspect` JSON accepts both object and array shapes and reports empty output with a clear error
- CI pins Go 1.26 across all modules so gosec runs, and coverage is written to `coverage.txt`
- Unit-test coverage raised above 90% with explicit testify assertions

## [0.2.0] - 2026-07-07

### Added
- Comprehensive benchmark suite comparing `applecontainer-go` vs `testcontainers-go`, covering driver operations, wait strategies and JSON parsing
- Issue and pull request templates to standardize community contributions
- GitNexus documentation and agent skill definitions for code intelligence integration

### Changed
- **Major refactor:** purged over-engineered lifecycle hooks and wait abstractions in favor of simpler, more direct patterns
- Replaced `github.com/goccy/go-json` with the standard library `encoding/json` to reduce dependencies
- Replaced custom log infrastructure with a thin `log/slog` wrapper
- Isolated the `examples` folder into its own Go module
- Integrated Codecov into the GitHub Actions CI pipeline
- Standardized testdata formatting and added a `Makefile` for local workflows
- Updated GitHub Actions versions and the coverage badge to track the `main` branch
- Updated the test-coverage output path and removed atomic mode

### Removed
- `ForExposedPort` and `ForMappedPort` port aliases
- Empty `PullOption`, `TerminateOption`, dead `SessionID()` and `Close()`
- No-op public functions `WithEnvFile`, `WithLogger` and `WithAdditionalLifecycleHooks`
- Unused `StatsEntry`, `ListEntry`, `parseList` and `parseStats`

### Fixed
- Resolved gosec security warnings across the codebase and updated CI workflows
- Fixed the `Makefile` benchmark target and Docker skip conditions

### Security
- `SECURITY.md` defining the security policy and vulnerability reporting process

### Documentation
- `CODE_OF_CONDUCT.md` establishing community guidelines
- Comprehensive contribution guidelines in `CONTRIBUTING.md`
- `CHANGELOG.md` tracking release history
- Updated architecture documentation and removed outdated plans

## [0.1.0] - 2026-06-17

This is the initial release of `applecontainer-go`, a lightweight `testcontainers-go`-style Go library designed to run Linux containers on macOS via Apple's native container orchestration engine.

### Added
- Native Apple virtualization integration for container lifecycles (`start`, `stop`, `kill`, `delete`, `copy`, `exec`)
- Flexible container customizer options (`WithExposedPorts`, `WithHostPortMapping`, `WithEnv`, `WithCPUs`, `WithMemory`, ...)
- Complete set of wait strategies:
  - **Listening Port**: checks that a port is open
  - **HTTP**: performs HTTP requests and validates status codes/body
  - **Log Stream**: scans stdout/stderr for a substring or regex
  - **Exec Command**: runs a command inside the container repeatedly
  - **SQL Database**: verifies database availability through `database/sql`
  - **File Check**: verifies a file exists inside the container
  - **Container Health**: waits for the container state to be running
  - **Exit Code**: waits for the container to exit
  - **Composite**: `ForAll` and `ForAny` combinations
- Volume and network creation and management APIs (`NewVolume`, `NewNetwork`)
- Interactive callback injection via `ContainerLifecycleHooks`
- `LogConsumer` interface for consuming container log streams
- Integration and unit test suites
- README documentation
