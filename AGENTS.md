# Repository Guidelines

## Project Overview

`applecontainer-go` is a `testcontainers-go`-style Go library for spinning up Linux containers through the **Apple native `container` CLI** on Apple Silicon macOS (26+). No Docker API, no daemon socket: every operation is an `os/exec` of the `container` binary, and all output is parsed from JSON or exit codes.

Module: `github.com/lynicis/applecontainer-go`; package name: `applecontainer`.

## Architecture & Data Flow

Root package `applecontainer`; three subpackages: `wait/` (readiness strategies), `log/` (slog wrappers), plus separate Go modules `benchmarks/` and `examples/`.

```
Run(ctx, img, opts...)            applecontainer.go
  └─ ContainerCustomizer → *ContainerRequest   options.go
  └─ req.Validate()                             container.go
  └─ Config.Read() + VersionCheck (retried)     config.go
  └─ Provider                                    provider.go
       └─ buildCreateArgs → []string             args.go
       └─ commandRunner.Run/Start                cli.go  ← only exec seam
       └─ container create → start → inspect JSON
  └─ waitTarget + wait.Strategy.WaitUntilReady  wait/
```

- `Run` (`applecontainer.go:48`) is the one-shot entry point; `ContainerRequest` (`container.go`) is the declarative spec; `Provider` (`provider.go`) is the CLI verb set; `Container` is a thin facade delegating everything to `Provider`.
- `Run` builds pull flags via `buildPullArgs` (`args.go`); any failure after `create` terminates the container, runs `req.Cleanups`, and returns a nil `*Container`.
- `commandRunner` (`cli.go:11`) is the **single seam to the `container` binary**. Production impl is `execRunner`; tests inject a fake. Preserve this boundary — never call `os/exec` directly from feature code.
- `providerRunnerOverride` (`provider.go:29`) lets tests swap the runner for `Run()` itself.
- Config is read once (`sync.Once`) from `~/.applecontainer.properties`, then overridden by `CONTAINER_BINARY`. `Reset()` exists for tests only and also clears the cached version check.
- Networking has two modes: direct container IP (default, `Host()` = container IP) and host-port mapping (`WithHostPortMapping(true)`, ephemeral ports bound on 127.0.0.1). Port/host resolution goes through `Inspect` JSON — see `Container.Host/Endpoint/MappedPort` (`container.go:173-231`).
- Concurrency: `sync.Once` for config and session id; the version check uses a mutex-guarded flag so a failed check is retried on the next `Run`; `atomic.Bool` for the running flag; goroutine + `io.MultiWriter` for log following.

## Key Directories

| Path | Purpose |
| :--- | :--- |
| `.` (root) | Library: container lifecycle, options, networking, volumes, inspect decoding |
| `wait/` | `wait.Strategy` interface + `ForListeningPort`, `ForHTTP`, `ForLog`, `ForExec`, `ForSQL`, `ForFile`, `ForHealth`, `ForExit`, `ForAll`, `ForAny` |
| `log/` | `slog` default logger + `log.TestLogger(t)` discard logger |
| `benchmarks/` | **Separate Go module**; head-to-head vs `testcontainers-go` |
| `examples/` | **Separate Go module**; `//go:build integration` end-to-end tests |
| `testdata/` | `inspect*.json` fixtures for parser tests |

## Development Commands

```bash
make test            # go test -race ./...                  (hermetic, no container runtime needed)
make test-coverage   # go test -v -race -coverprofile=coverage.txt ./...
make lint            # go tool golangci-lint run ./...
make fmt             # go tool golangci-lint fmt ./...
make sec             # go tool gosec ./...
make test-examples   # cd examples && APPLECONTAINER_INTEGRATION=1 go test -tags integration -v ./...
make test-benchmark  # cd benchmarks && APPLECONTAINER_BENCHMARK=1 go test -bench=. -benchtime=1x -tags benchmark -timeout=600s ./...
```

`golangci-lint` and `gosec` are pinned in the `tool` block of `go.mod` (`go.mod:257`); the Makefile invokes them through `$(GOLANGCI_LINT)` / `$(GOSEC)`, which expand to `go tool ...`, so no `PATH` install is needed.

## Code Conventions & Common Patterns

- **Functional options**: `ContainerCustomizer func(*ContainerRequest) error`. `WithXxx` mutates the request; maps are *merged*, slices *appended*. `ProcessOption` (in `exec.go`) is the exec-only equivalent.
- **Errors**: `fmt.Errorf("applecontainer: <verb> ...: %w", err)` — always wrapped, lower-case after the prefix. `runError` (`cli.go:69`) carries bin/args/exit code/stderr. No exported sentinel errors; not-found detection is string matching (`isNotFoundError`).
- **Context**: first parameter of every public method; passed to `exec.CommandContext` so cancellation kills the CLI. Only log followers and `t.Cleanup` use `context.Background()`.
- **Logging**: use the internal `log` package (`log.Printf`, `log.Default()`), never `fmt.Println`. Per-instance `*slog.Logger` fields exist but are mostly vestigial.
- **Temp files**: cidfile and copy staging use `os.CreateTemp` with deferred cleanup and `checkedFileMode` for host modes.
- **JSON**: struct tags are camelCase to match the CLI schema; the inspect root type is `Configuration` (renamed to avoid clashing with library `Config`). Add fixtures in `testdata/` and parse tests in `inspect_test.go`.
- **Naming**: `WithXxx` options, `NewXxx` resource constructors, `GetContainerID`/`Name`/`Host`/`Endpoint` getters; internal helpers lowerCamel (`buildCreateArgs`, `parseInspect`).
- Formatting is `gofmt` + `goimports`; lint enables `misspell` (US locale) on top of golangci-lint defaults.
- Commits: imperative mood, summary < 50 chars, blank line then body; reference issues. PR template at `.github/pull_request_template.md`.

## Testing & QA

- Root tests are **hermetic**: `fakeRunner` (`config_test.go:14`) implements `commandRunner` and is installed via `providerRunnerOverride` or `&Provider{runner: fake, ...}`. Fake side effects explicitly (e.g. write the cidfile in the `create` test).
- Style is mixed: stdlib `testing` with `t.Fatalf` in most files; `testify` (`assert`/`require`) in `container_test.go`, `provider_test.go`, `cli_test.go`, `applecontainer_test.go`, `wait/wait_test.go`. Use `require` for preconditions, `assert` for value checks; table-driven subtests where cases multiply (`container_request_test.go`).
- `wait/` uses `fakeTarget` implementing `StrategyTarget` plus `httptest.Server`, `net.Listen`, and a registered `mockDriver` for SQL — no runtime needed.
- Anything touching the real CLI belongs in `examples/` behind `//go:build integration` + `APPLECONTAINER_INTEGRATION=1`, and must call `applecontainer.SkipIfProviderNotHealthy(t)` so unsupported machines skip rather than fail. Use `CleanupContainer` / `CleanupNetwork` for teardown.
- Benchmarks live in the separate `benchmarks/` module, gated by `APPLECONTAINER_BENCHMARK=1`; `RunWithBoth` runs each case against both runtimes.
- CI (`.github/workflows/ci.yml`): lint + gosec on ubuntu, `make test-coverage` on `macos-latest`, coverage uploaded to Codecov. Integration tests and benchmarks are not in CI.
- `Reset()` also resets the version-check state, so tests that exercise `Run`'s version gate must call it first.
- New behavior needs a unit test; new `WithXxx` options or wait strategies also need a README row and, where practical, an `examples/` case.
- Add user-visible changes under `## [Unreleased]` in `CHANGELOG.md` (Keep a Changelog + SemVer sections: Added/Changed/Removed/Fixed/Security/Documentation).
- Gotcha: `make test-coverage` writes `coverage.txt`, but `.gitignore` only lists `coverage.out`/`coverage.html` — do not commit the profile.

## Runtime/Tooling Preferences

- Go **1.27** (`go.mod` is the source of truth; `CONTRIBUTING.md` still says 1.26).
- Runtime target: macOS 26+, Apple Silicon, `container` CLI installed. Unit tests deliberately do not require it.
- No package manager beyond `go`; no code generation, no build tags in the root package.
- `make vuln-check` is referenced by `CONTRIBUTING.md` and the PR template but **does not exist** in `Makefile` — use `make sec` and `govulncheck ./...` manually.

## Important Files

| File | Role |
| :--- | :--- |
| `applecontainer.go` | `Run`, `Prune`, `SessionID`, version-check gate |
| `container.go` | `ContainerRequest`, `Container`, `Validate`, port/host resolution, `waitTarget` |
| `provider.go` | CLI verb implementations, `Provider`, runner override |
| `args.go` | `buildCreateArgs` — request → CLI flags |
| `cli.go` | `commandRunner` seam, `execRunner`, `runError` |
| `config.go` | `Config`, `Read`, `VersionCheck`, env/properties precedence |
| `options.go` / `exec.go` | Functional options |
| `inspect.go` | JSON schema of `container inspect` |
| `Makefile`, `.golangci.yml`, `.github/workflows/ci.yml` | Build, lint, CI gates |
