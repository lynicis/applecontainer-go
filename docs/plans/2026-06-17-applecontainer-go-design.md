# applecontainer-go Design

**Status:** Approved 2026-06-17
**Goal:** A testcontainers-go-style Go library that spins up Apple Container (`container` CLI) Linux containers as test dependencies, with wait strategies, lifecycle hooks, and build-from-file.

---

## 1. Module & package layout

Module `github.com/lynicis/applecontainer-go`, root package `applecontainer`. Stdlib-only, no CGo.

```
applecontainer-go/
  go.mod
  applecontainer.go      Run(), CleanupContainer(), SessionID(), Prune(), VersionCheck()
  container.go           Container interface, ContainerRequest, FromContainerfile, ContainerFile
  provider.go            ContainerProvider interface + CLIProvider (shells out)
  cli.go                 commandRunner seam (the only testable boundary to the binary)
  args.go                ContainerRequest -> CLI arg list (pure function)
  inspect.go             Inspect/State/NetworkInfo JSON types + parsing
  build.go               FromContainerfile -> container build
  options.go             ContainerCustomizer, CustomizeRequestOption, all With* options
  lifecycle.go           ContainerLifecycleHooks + reflection-ordered combination
  network.go             Network, NetworkProvider, NetworkRequest
  volume.go              Volume, VolumeProvider, VolumeRequest
  logconsumer.go         LogConsumer interface + long-lived `logs -f` fan-out
  config.go              Config singleton (env + ~/.applecontainer.properties)
  testing.go             CleanupContainer/CleanupNetwork/SkipIfProviderNotHealthy
  exec.go                ProcessOption for Exec (WithUser/WithWorkingDir/WithEnv/Multiplexed)
  log/
    logger.go            log.Logger interface, Default(), TestLogger(t)
  wait/
    wait.go              Strategy, StrategyTarget, StrategyTimeout interfaces
    all.go, any.go       ForAll / ForAny + .WithDeadline
    log.go, http.go      ForLog / ForHTTP
    exec.go, exit.go     ForExec / ForExit
    port.go              ForListeningPort/ForExposedPort/ForMappedPort (adapted to IP model)
    health.go, sql.go    ForHealth / ForSQL
    file.go              ForFile
  examples/              nginx_test.go, redis_test.go (integration-tagged)
  docs/plans/            this file + the implementation plan
```

## 2. Integration path - shell out to the `container` CLI

The Go library talks to Apple Container exclusively via the `container` CLI (v1.0.0+), the only documented integration surface. One seam, fully unit-testable:

```go
type commandRunner interface {
    Run(ctx context.Context, args []string, stdin []byte) (stdout, stderr []byte, exitCode int, err error)
    Start(ctx context.Context, args []string, stdin io.Reader) (cmd *exec.Cmd, stdout, stderr io.Reader, err error)
}
```

Tactics:
- Long-lived `logs -f` per container with fan-out to wait strategies and user LogConsumers.
- `inspect` polling at `wait.PollInterval` (100ms default) with per-tick cache.
- `--rm` + session labels on every container.
- `container --version` gate on first use (require >= 1.0.0).
- `--progress plain` for build/pull routed to the logger.

Rejected alternatives:
- XPC to `container-apiserver` directly: private/undocumented schema, breaks on every `container` bump, requires CGo.
- Swift `Containerization` package via CGo/Swift bridge: requires Swift toolchain, CGo interop pain, only patch-version stable.

## 3. Core types (Apple-native, no moby dependency)

### ContainerRequest

Declarative struct. Apple-specific knobs replace Docker's `HostConfigModifier`/`ConfigModifier`:

| Field group | Fields |
|---|---|
| Image | `Image`, `FromContainerfile`, `AlwaysPull`, `Platform`, `Arch`, `OS` |
| Process | `Cmd`, `Entrypoint`, `Env`, `WorkingDir`, `User`, `Init` |
| Network | `ExposedPorts []string`, `HostPorts map[string]int` (opt-in), `Networks`, `NetworkAliases`, `DNS`, `DNSDomain`, `DNSSearch`, `NoDNS` |
| Storage | `Volumes []VolumeMount`, `Mounts []Mount`, `Tmpfs`, `ShmSize`, `ReadOnlyRootfs`, `Files []ContainerFile` |
| Resources | `CPUs`, `Memory`, `CapAdd`, `CapDrop`, `Ulimits` |
| Apple | `Rosetta`, `Name` (= container ID) |
| Meta | `Labels`, `WaitingFor`, `LifecycleHooks`, `LogConsumerCfg`, `HostPortMapping bool` |
| Escape hatch | `CLIArgsModifier func([]string) []string` (replaces moby's `HostConfigModifier`) |

### Container interface

```go
type Container interface {
    GetContainerID() string
    Endpoint(ctx context.Context, port string) (string, error)
    PortEndpoint(ctx context.Context, port string, proto string) (string, error)
    Host(context.Context) (string, error)
    MappedPort(ctx context.Context, port string) (int, error)
    ContainerIP(context.Context) (string, error)
    Inspect(context.Context) (*Inspect, error)
    State(context.Context) (*State, error)
    IsRunning() bool
    SessionID() string
    Start(context.Context) error
    Stop(context.Context, *time.Duration) error
    Terminate(ctx context.Context, opts ...TerminateOption) error
    Logs(context.Context) (io.ReadCloser, error)
    Exec(ctx context.Context, cmd []string, opts ...ProcessOption) (int, []byte, error)
    CopyToContainer(ctx context.Context, content []byte, containerPath string, mode int64) error
    CopyFileToContainer(ctx context.Context, hostPath, containerPath string, mode int64) error
    CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error)
    Networks(context.Context) ([]string, error)
}
```

Return types are our own `*Inspect`/`*State` (parsed JSON), never moby's.

### Inspect JSON types

Mirror `container inspect` output: `id`, `image`, `state{status,running,exitCode,oomKilled,startedAt,finishedAt,error}`, `networks[]{name,ipv4Address,ipv6Address,mac}`, `labels`, `config{env,cmd,entrypoint,workingDir}`. Exact field names locked down by running one real container during implementation (schema captured from container v1.0.0).

## 4. Options pattern - ported verbatim from testcontainers-go

```go
type ContainerCustomizer interface { Customize(req *ContainerRequest) error }
type CustomizeRequestOption func(req *ContainerRequest) error
```

Full `With*` set mirrors testcontainers-go (basic, networking, resources, mounts, files, build, lifecycle, logging, wait, platform, advanced) plus Apple additions: `WithCPUs`, `WithMemory`, `WithRosetta`, `WithReadOnlyRootfs`, `WithInit`, `WithHostPortMapping()` (the opt-in toggle), `WithCLIArgsModifier`. Request merging via a ~40-line `mergeRequest` (append slices, override scalars, merge maps) - no `mergo` dep.

## 5. Wait strategies - ported near-verbatim

The `wait/` package only depends on `StrategyTarget`, so it's the most reusable subsystem. Ported with one adaptation: `StrategyTarget.MappedPort` returns `int` and `Inspect`/`State` are our types.

- `ForListeningPort("5432/tcp")` dials the container IP:port directly (no internal/external split - same network from host's view in Apple's model).
- `ForHealth` is adapted: Apple inspect has no `HEALTHCHECK` field, so it checks `State.Status == "running"` + no exit, with a documented option to exec the image's healthcheck command via `ForExec`.
- `ForLog`/`ForHTTP`/`ForExec`/`ForExit`/`ForSQL`/`ForFile`/`ForAll`/`ForAny` + deadlines - identical to upstream.

## 6. Lifecycle hooks - ported verbatim

`ContainerLifecycleHooks` with `PreBuilds/PostBuilds/PreCreates/PostCreates/PreStarts/PostStarts/PostReadies/PreStops/PostStops/PreTerminates/PostTerminates`. Default-pre -> user-pre -> user-post -> default-post ordering via reflection.

Default hooks:
- logging (emoji-tagged phase logs)
- pre-create (request -> arg list, `--rm`, labels, `--cidfile`)
- build (if `FromContainerfile`)
- copy-files (PostCreate via `container cp`)
- log-consumers (start/stop the `logs -f` fan-out)
- readiness (PostStart -> `WaitingFor.WaitUntilReady` -> set `isRunning`)

## 7. Networking & ports - two modes

### Direct IP (default)
No `--publish`; container gets `192.168.64.x`. `Host()` -> container IP, `MappedPort("5432")` -> `5432`, `Endpoint("5432")` -> `192.168.64.9:5432`. Zero port conflicts -> perfect for `t.Parallel()`.

### Host port mapping (opt-in `WithHostPortMapping()`)
Allocate ephemeral host port via `net.Listen("127.0.0.1:0")`, pass `--publish <ephemeral>:<containerPort>`. `Host()` -> `localhost`, `MappedPort` -> ephemeral. Needed only when test code itself runs in a container or needs `localhost:<port>`.

Networks: `NewNetwork()` wraps `container network create`/`inspect`/`delete`. Container-to-container by IP on macOS 26; DNS-by-name requires a configured `container system dns` domain (documented, not auto-configured in v1).

## 8. Cleanup - simple, no reaper

`--rm` + `-l applecontainer=true -l applecontainer.session=<sessionID>` on every container. `SessionID()` from parent PID + creation time (same trick as testcontainers-go -> `go test ./...` packages share a session). `CleanupContainer(t, ctr)` -> `t.Cleanup` -> idempotent `Terminate` (swallows not-found/already-removed). `Prune(ctx)` -> `container prune` for manual sweeps. No sidecar, no goroutine, no Docker socket.

## 9. Build-from-file

`WithContainerfile(FromContainerfile{Context, File, BuildArgs, Tags, Target, Platform, NoCache, Pull, Secrets, KeepImage})` -> `defaultBuildHook` runs `container build -t <tag> --progress plain <ctx>`, sets `req.Image = <tag>`. `KeepImage` controls whether `Terminate` removes the built image. Apple's BuildKit handles Dockerfile/Containerfile, multi-stage, build-args, secrets natively.

## 10. Config

```go
type Config struct {
    BinaryPath      string
    Debug           bool
    DefaultNetwork  string // "default"
    DefaultPlatform string // env CONTAINER_DEFAULT_PLATFORM
    HubImagePrefix  string // env APPLECONTAINER_HUB_IMAGE_NAME_PREFIX
    PullTimeout     time.Duration
}
```

Read once (`sync.Once`) from `~/.applecontainer.properties` + env (`APPLECONTAINER_*`, `CONTAINER_DEBUG`, `CONTAINER_DEFAULT_PLATFORM`). `config.Read()`/`config.Reset()` for tests.

## 11. Library self-testing

- **Unit (default):** fake `commandRunner` returns canned JSON, records arg lists. Tests arg-building, JSON parsing, option merge, hook order, wait strategies, endpoint math, session ID, labels, port allocation. No binary required.
- **Integration (`//go:build integration`, gated on `APPLECONTAINER_INTEGRATION=1`):** real `container` CLI - nginx smoke, postgres container pgx connect, parallel containers, ForLog/ForPort/ForHTTP/ForSQL, build-from-file, networks, volumes. Requires macOS 26 + Apple silicon + `container system start`.
- `SkipIfProviderNotHealthy(t)` checks `container --version` + `container system status`.

## 12. Dependencies & compatibility

Stdlib only. `database/sql` used only in `wait/sql.go` (driver supplied by user). No `mergo`, no moby, no CGo. Requires `container` >= 1.0.0, macOS 26, Apple silicon - `VersionCheck()` on first use with a clear error. Pre-1.0: stability within patch versions (mirrors Apple's stance).

## 13. Explicitly OUT of v1

No modules (such as a postgres module), no `compose`, no `modulegen`, no SSHD host-port-access (Apple's direct IP makes it unnecessary - containers reach the host at the vmnet gateway `192.168.64.1`), no reaper process, no `ImageSubstitutor` interface (just `HubImagePrefix`), no `--publish-socket`/`--ssh`/`--virtualization`/`--runtime`/`--kernel` wrappers (reachable via `WithCLIArgsModifier`). All layer cleanly on top later.

## Appendix: Apple Container facts (verified 2026-06-17)

- **Architecture:** `container` CLI -> `container-apiserver` (launch agent) -> XPC helpers (`container-core-images`, `container-network-vmnet`, per-container `container-runtime-linux`). One lightweight VM per container, OCI images, Virtualization.framework + vmnet.
- **Programmability:** Only the Swift `Containerization` package is documented for programmatic use. No Go SDK, no public gRPC/socket API. -> A Go library must shell out to the `container` CLI.
- **Networking:** Each container gets its own IP on `192.168.64.0/24` (vmnet), directly reachable from the host. Container-to-container works on macOS 26. `--publish [host-ip:]host-port:container-port[/protocol]` for host port mapping.
- **Lifecycle:** `create` (stopped) -> `start` -> `run` (create+start) -> `stop` (graceful, 5s default) -> `kill` -> `delete` (`--force` if running) -> `prune`. `--rm` auto-removes on stop. `--cidfile` records ID. Labels (`-l`) supported.
- **JSON outputs:** `list/stats/image ls/network ls/volume ls` support `--format json|yaml|toml`. `inspect`/`image inspect` output JSON by default.
- **Limitations:** Partial memory ballooning (freed pages not returned to host - matters for parallel tests); macOS 15 has broken container-to-container networking (require macOS 26).
- **CLI version:** `container version 1.0.0 (build: release, commit: unspeci)`.
