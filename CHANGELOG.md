# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Created [SECURITY.md](file:///Users/lynicis/Projects/applecontainer-go/SECURITY.md) to define security policy and vulnerability reporting guidelines.
- Created `CODE_OF_CONDUCT.md` to establish community guidelines.
- Added comprehensive contribution guidelines in [CONTRIBUTING.md](file:///Users/lynicis/Projects/applecontainer-go/CONTRIBUTING.md).

### Changed
- Integrated Codecov into the GitHub Actions CI pipeline and updated badge/coverage reporting.
- Standardized testdata formatting and added a `Makefile` to simplify local workflows.
- Isolated/separated the `examples` folder into its own Go module.

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
