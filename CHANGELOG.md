# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
