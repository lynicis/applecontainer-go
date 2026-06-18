# applecontainer-go Implementation Plan

**Goal:** Build `github.com/lynicis/applecontainer-go`, a testcontainers-go-style Go library that spins up Apple Container (`container` CLI) Linux containers as test dependencies, with wait strategies, lifecycle hooks, and build-from-file.

**Architecture:** Shell out to the `container` CLI (v1.0.0+) via a single `commandRunner` seam. Apple-native types throughout (no moby dependency). Direct container IP by default, opt-in host port mapping. `--rm` + session labels + `t.Cleanup` for teardown. Port the runtime-agnostic subsystems (wait strategies, options, hooks) from testcontainers-go near-verbatim; rewrite the Docker-specific parts (provider, reaper->simple-cleanup, port model).

**Tech Stack:** Go 1.26, stdlib only (no CGo, no moby, no mergo). Apple `container` CLI >= 1.0.0, macOS 26, Apple silicon. TDD: unit tests with a fake `commandRunner` (default build); integration tests behind `//go:build integration` + `APPLECONTAINER_INTEGRATION=1`.

---

## Phase 0 - Worktree & docs

### Task 0.1: Create worktree + design doc
**Files:**
- Create: `docs/plans/2026-06-17-applecontainer-go-design.md`
- Create: `docs/plans/2026-06-17-applecontainer-go-plan.md`
**Step 1:** `git worktree add ../applecontainer-go-impl -b feat/core-library`
**Step 2:** Write both docs (content = approved design + this plan).
**Step 3:** `git add docs/ && git commit -m "docs: add applecontainer-go design and implementation plan"`

> Tasks 0.x run once plan mode is exited. All subsequent tasks execute in the worktree.

---

## Phase 1 - Foundation: the CLI seam

The `commandRunner` interface is the only boundary to the `container` binary and the only thing unit tests fake. Get it right first; everything builds on it.

### Task 1.1: `cli.go` - commandRunner seam
**Files:**
- Create: `cli.go`
- Create: `cli_test.go`

**Step 1: Write the failing test**
```go
// cli_test.go
package applecontainer

import (
	"context"
	"testing"
)

func TestExecRunnerReturnsStdout(t *testing.T) {
	r := newExecRunner("echo")
	out, stderr, code, err := r.Run(context.Background(), []string{"hi"}, nil)
	if err != nil { t.Fatalf("err: %v", err) }
	if code != 0 { t.Fatalf("code=%d stderr=%s", code, stderr) }
	if string(out) != "hi\n" { t.Fatalf("out=%q", out) }
}

func TestExecRunnerPassesStdin(t *testing.T) {
	r := newExecRunner("cat")
	out, _, code, err := r.Run(context.Background(), nil, []byte("ping"))
	if err != nil || code != 0 { t.Fatalf("err=%v code=%d", err, code) }
	if string(out) != "ping" { t.Fatalf("out=%q", out) }
}

func TestExecRunnerPropagatesExitCode(t *testing.T) {
	r := newExecRunner("sh")
	_, _, code, err := r.Run(context.Background(), []string{"-c", "exit 7"}, nil)
	if err == nil { t.Fatal("want error for non-zero exit") }
	if code != 7 { t.Fatalf("code=%d want 7", code) }
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./... -run TestExecRunner`
Expected: FAIL with "undefined: newExecRunner"

**Step 3: Write minimal implementation**
```go
// cli.go
package applecontainer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// commandRunner is the single seam to the `container` binary. Tests fake it.
type commandRunner interface {
	Run(ctx context.Context, args []string, stdin []byte) (stdout, stderr []byte, exitCode int, err error)
	Start(ctx context.Context, args []string, stdin io.Reader) (cmd *exec.Cmd, stdout, stderr io.Reader, err error)
}

// execRunner implements commandRunner via os/exec against a fixed binary path.
type execRunner struct{ bin string }

func newExecRunner(bin string) *execRunner { return &execRunner{bin: bin} }

func (r *execRunner) Run(ctx context.Context, args []string, stdin []byte) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, r.bin, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		code = -1
	}
	if code != 0 {
		err = &runError{bin: r.bin, args: args, code: code, stderr: errb.String(), cause: err}
	}
	return out.Bytes(), errb.Bytes(), code, err
}

func (r *execRunner) Start(ctx context.Context, args []string, stdin io.Reader) (*exec.Cmd, io.Reader, io.Reader, error) {
	cmd := exec.CommandContext(ctx, r.bin, args...)
	cmd.Stdin = stdin
	outPR, outPW := io.Pipe()
	errPR, errPW := io.Pipe()
	cmd.Stdout, cmd.Stderr = outPW, errPW
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	return cmd, outPR, errPR, nil
}

type runError struct {
	bin   string
	args  []string
	code  int
	stderr string
	cause error
}

func (e *runError) Error() string {
	return fmt.Sprintf("%s %s: exit %d: %s", e.bin, strings.Join(e.args, " "), e.code, e.stderr)
}
func (e *runError) Unwrap() error { return e.cause }
```

**Step 4: Run test to verify it passes**
Run: `go test ./... -run TestExecRunner`
Expected: PASS

**Step 5: Commit**
```bash
git add cli.go cli_test.go
git commit -m "feat: add commandRunner seam for container CLI"
```

### Task 1.2: `config.go` - Config singleton + binary discovery + version gate
**Files:**
- Create: `config.go`
- Create: `config_test.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Key code (config.go):**
```go
type Config struct {
	BinaryPath      string
	Debug           bool
	DefaultNetwork  string // "default"
	DefaultPlatform string // env CONTAINER_DEFAULT_PLATFORM
	HubImagePrefix  string // env APPLECONTAINER_HUB_IMAGE_NAME_PREFIX
	PullTimeout     time.Duration
}

func Read() Config          // sync.Once singleton; ~/.applecontainer.properties + env
func Reset()                // test-only
func (c Config) runner() commandRunner { return newExecRunner(c.BinaryPath) }
```

**Version gate:**
`VersionCheck(ctx) (string, error)` runs `container --version`, parses `container version 1.0.0 (build: release, commit: unspeci)`, returns version, errors clearly if < 1.0.0 or binary missing. Called lazily on first `Run()`.

**Step 1: Write failing tests (fake runner)**
Assert env overrides properties; assert `VersionCheck` parses the real format and rejects `0.9.0`; assert missing binary errors with a helpful message.

**Step 2:** Run -> FAIL
**Step 3:** Implement `Config`, `Read`, `Reset`, `VersionCheck`.
**Step 4:** Run -> PASS
**Step 5:** Commit `feat: add config singleton and version gate`

### Task 1.3: `log/logger.go` - minimal logger interface
**Files:**
- Create: `log/logger.go`
- Create: `log/logger_test.go`

Port testcontainers-go's `log.Logger` interface (`Printf(format, args)`), `log.Default()`, `log.TestLogger(t)`. ~40 lines.

**Step 1:** Test `TestLogger` writes via `t.Logf`.
**Step 2-5:** Implement + pass + commit `feat: add log package`.

---

## Phase 2 - Inspect JSON types

### Task 2.1: `inspect.go` - Inspect/State/NetworkInfo types + parsing
**Files:**
- Create: `inspect.go`
- Create: `inspect_test.go`
- Create: `testdata/inspect.json`

**Implementation note:** Before writing the structs, run one real container to capture the exact JSON:
```bash
container run -d --name probe nginx:latest
container inspect probe > testdata/inspect.json
container stop probe && container rm probe
```
Lock the struct field names + json tags to that. Add a `// schema captured from container v1.0.0` comment.

**Step 1: Write failing test**
```go
func TestParseInspectRoundTrip(t *testing.T) {
	data, err := os.ReadFile("testdata/inspect.json")
	if err != nil { t.Fatal(err) }
	got, err := parseInspect(data)
	if err != nil { t.Fatalf("parseInspect: %v", err) }
	if got.ID == "" { t.Fatal("empty ID") }
	if len(got.Networks) == 0 { t.Fatal("no networks") }
	if got.Networks[0].IPv4Address == "" { t.Fatal("empty ipv4") }
}
```

**Step 2:** Run -> FAIL (undefined: parseInspect)
**Step 3:** Implement structs + `parseInspect` (uses `encoding/json`; ignore unknown fields, validate required ones).
**Step 4:** Also parse `container list --format json` (`[]listEntry`) and `container stats --format json --no-stream`.
**Step 5:** Commit `feat: add inspect/list/stats JSON types`.

---

## Phase 3 - Provider (the shell-out engine)

### Task 3.1: `provider.go` - ContainerProvider interface + CLIProvider
**Files:**
- Create: `provider.go`
- Create: `provider_test.go`

```go
type ContainerProvider interface {
	CreateContainer(ctx context.Context, req *ContainerRequest) (*cliContainer, error)
	StartContainer(ctx context.Context, c *cliContainer) error
	StopContainer(ctx context.Context, id string, timeout *time.Duration) error
	KillContainer(ctx context.Context, id string, signal string) error
	DeleteContainer(ctx context.Context, id string, force bool) error
	InspectContainer(ctx context.Context, id string) (*Inspect, error)
	ContainerLogs(ctx context.Context, id string, follow bool, n int) (io.ReadCloser, error)
	ExecContainer(ctx context.Context, id string, cmd []string, opts ...ProcessOption) (int, []byte, error)
	CopyToContainer(ctx context.Context, id, containerPath string, content []byte, mode int64) error
	CopyFileFromContainer(ctx context.Context, id, path string) (io.ReadCloser, error)
	ImagePull(ctx context.Context, ref string, opts ...PullOption) error
	ImageInspect(ctx context.Context, ref string) (*ImageInspect, error)
	Health(ctx context.Context) error
	Close() error
}

type cliProvider struct {
	runner commandRunner
	cfg    Config
	log    log.Logger
}
```

**Tests (fake runner):** each method asserts the exact arg list it builds and parses canned JSON. These tests ARE the spec for arg-building. Implement method-by-method, TDD:

- `CreateContainer` -> builds `create` args from `ContainerRequest` (see Task 4.3 for arg-builder), calls `container create`, reads `--cidfile` or stdout for the ID.
- `StartContainer` -> `container start <id>`.
- `StopContainer` -> `container stop --time <s> <id>` (nil timeout = 5).
- `InspectContainer` -> `container inspect <id>` -> `parseInspect`.
- `ContainerLogs` -> `Start` a long-lived `container logs -f <id>` (follow) or `Run` `container logs -n <n> <id>`.
- `ExecContainer` -> `container exec <flags> <id> <cmd...>` -> capture exit code.
- `CopyToContainer` -> write bytes to a temp file, `container cp <tmp> <id>:<path>`.
- `CopyFileFromContainer` -> `container cp <id>:<path> <tmp>`, read tmp.
- `ImagePull` -> `container image pull --progress plain <ref>`.
- `Health` -> `VersionCheck` + `container system status` parse.

**Commit per method** (`feat: provider create`, `feat: provider stop`, ...). ~8 commits.

### Task 3.2: `container.go` - Container interface + cliContainer impl
**Files:**
- Create: `container.go`
- Create: `container_test.go`

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

type cliContainer struct {
	provider  *cliProvider
	id        string
	image     string
	req       ContainerRequest
	log       log.Logger
	isRunning atomic.Bool
	lifecycle []ContainerLifecycleHooks
	logFanout *logFanout
}
```

**Tests (fake provider):** `Endpoint` math for both IP and host-port modes; `IsRunning` after `Start`; `Terminate` is idempotent (second call swallows not-found).
**Commit:** `feat: add Container interface and cliContainer`

---

## Phase 4 - Request, options, arg-builder

### Task 4.1: `container.go` - ContainerRequest + FromContainerfile + ContainerFile
**Files:**
- Modify: `container.go`
- Create: `container_request_test.go`

Define the `ContainerRequest` struct (field table from design section 3), `FromContainerfile`, `ContainerFile`. `Validate()` method (no both Image+FromContainerfile; no duplicate mount targets; host-port mode requires ExposedPorts).
**Commit:** `feat: add ContainerRequest and FromContainerfile`

### Task 4.2: `options.go` - ContainerCustomizer + With* options + mergeRequest
**Files:**
- Create: `options.go`
- Create: `options_test.go`

Port testcontainers-go's options pattern. Key: `ContainerCustomizer` interface + `CustomizeRequestOption` func type. Implement `mergeRequest(dst, src)` (append slices, override scalars, merge maps - ~40 lines, no mergo).

Full `With*` set:
- **Basic:** `WithImage`, `WithExposedPorts`, `WithEnv`, `WithEntrypoint`, `WithEntrypointArgs`, `WithCmd`, `WithCmdArgs`, `WithLabels`, `WithWaitingFor`/`WithWaitStrategy`/`WithWaitStrategyAndDeadline`/`WithAdditionalWaitStrategy`
- **Resources:** `WithCPUs`, `WithMemory`, `WithCapAdd`, `WithCapDrop`, `WithUlimits`
- **Process:** `WithWorkingDir`, `WithUser`, `WithInit`, `WithEnvFile`
- **Network:** `WithNetwork`, `WithNewNetwork`, `WithNetworkName`, `WithDNS`, `WithDNSDomain`, `WithDNSSearch`, `WithNoDNS`, `WithHostPortMapping`
- **Storage:** `WithMounts`, `WithVolumes`, `WithTmpfs`, `WithShmSize`, `WithReadOnlyRootfs`, `WithFiles`
- **Apple:** `WithRosetta`, `WithName`, `WithPlatform`, `WithArch`, `WithOS`, `WithAlwaysPull`
- **Build:** `WithContainerfile`
- **Lifecycle:** `WithLifecycleHooks`, `WithAdditionalLifecycleHooks`
- **Logging:** `WithLogConsumers`, `WithLogger`
- **Escape:** `WithCLIArgsModifier`

**Tests:** each `With*` applies correctly; merge order (defaults then user opts); `WithWaitStrategy` wraps in `wait.ForAll().WithDeadline(60s)`.
**Commit:** `feat: add ContainerCustomizer and With* options` (split into 2-3 commits by group).

### Task 4.3: `args.go` - ContainerRequest -> CLI arg list
**Files:**
- Create: `args.go`
- Create: `args_test.go`

Pure function `buildCreateArgs(req *ContainerRequest) []string` - the heart of the provider. Maps every `ContainerRequest` field to `container create` flags:

| Request field | CLI flag |
|---|---|
| `Image`/`Cmd` | positional + trailing args |
| `Env` | `-e k=v` (repeatable) |
| `WorkingDir` | `-w` |
| `User` | `-u` |
| `Init` | `--init` |
| `ExposedPorts` (IP mode) | no-op |
| `ExposedPorts` (host-port mode) | `--publish <ephemeral>:<port>` |
| `Networks` | `--network <name>` |
| `Volumes`/`Mounts` | `-v`/`--mount` |
| `Tmpfs` | `--tmpfs` |
| `ShmSize` | `--shm-size` |
| `ReadOnlyRootfs` | `--read-only` |
| `CPUs` | `-c` |
| `Memory` | `-m` |
| `CapAdd/Drop` | `--cap-add/--cap-drop` |
| `Rosetta` | `--rosetta` |
| `Name` | `--name` |
| `Labels` | `-l k=v` |
| (always) | `--rm` + `--cidfile <tmp>` + session labels |
| `Platform`/`Arch`/`OS` | `--platform`/`--arch`/`--os` |
| `Entrypoint` | `--entrypoint` |
| `CLIArgsModifier` | applied last |

Ephemeral host port allocation (host-port mode): `net.Listen("127.0.0.1:0")` -> close -> use the port.

**Tests:** table-driven, assert exact arg slices for representative requests (minimal, full, host-port mode, with network, with volume, with build tag).
**Commit:** `feat: add ContainerRequest to CLI arg builder`

---

## Phase 5 - Lifecycle hooks + Run entrypoint

### Task 5.1: `lifecycle.go` - hooks + reflection ordering
**Files:**
- Create: `lifecycle.go`
- Create: `lifecycle_test.go`

Port testcontainers-go's `ContainerLifecycleHooks` + `combineContainerHooks` (reflection over struct fields: default-pre -> user-pre -> user-post -> default-post).

```go
type ContainerRequestHook func(ctx context.Context, req *ContainerRequest) error
type ContainerHook func(ctx context.Context, c Container) error

type ContainerLifecycleHooks struct {
	PreBuilds, PostBuilds         []ContainerRequestHook
	PreCreates, PostCreates       []ContainerRequestHook
	PreStarts, PostStarts         []ContainerHook
	PostReadies                   []ContainerHook
	PreStops, PostStops           []ContainerHook
	PreTerminates, PostTerminates []ContainerHook
}
```

Default hooks:
- `defaultLoggingHook(logger)` - emoji-tagged phase logs.
- `defaultPreCreateHook` - invoke `buildCreateArgs`, allocate ephemeral ports (host-port mode), call `provider.CreateContainer`.
- `defaultBuildHook` - if `FromContainerfile`, run `container build -t <tag>` then set `req.Image`.
- `defaultCopyFilesHook` - PostCreate: `CopyToContainer` for each `req.Files`.
- `defaultLogConsumersHook` - PostStart: start `logFanout` (`logs -f`); PostStop: stop it.
- `defaultReadinessHook` - PostStart: `req.WaitingFor.WaitUntilReady(ctx, container)`; on success `isRunning.Store(true)`.

**Tests (fake provider):** assert hook execution order with recording hooks; assert default readiness hook sets isRunning; assert copy-files runs in PostCreate; assert build hook runs before create when `FromContainerfile` set.
**Commit:** `feat: add lifecycle hooks`

### Task 5.2: `applecontainer.go` - Run() entrypoint + SessionID + CleanupContainer + Prune
**Files:**
- Create: `applecontainer.go`
- Create: `applecontainer_test.go`

```go
func Run(ctx context.Context, img string, opts ...ContainerCustomizer) (*cliContainer, error) {
	req := &ContainerRequest{Image: img}
	for _, o := range opts {
		if err := o.Customize(req); err != nil {
			return nil, fmt.Errorf("customize: %w", err)
		}
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := versionCheckOnce(ctx); err != nil {
		return nil, err
	}
	provider := newCLIProvider(config.Read())
	c := &cliContainer{provider: provider, image: img, req: *req, log: defaultLogger(req), lifecycle: defaultHooks(req)}
	if err := c.executeLifecycle(ctx, true); err != nil {
		return c, err
	}
	return c, nil
}

func SessionID() string              // parent PID + creation time -> sha1("applecontainer-go:"+...)
func CleanupContainer(t testing.TB, c Container, opts ...TerminateOption)
func Prune(ctx context.Context) error
func GenericLabels() map[string]string  // applecontainer=true, .session=<id>
```

**Tests (fake provider via dependency injection):** `Run` applies options in order; default wait strategy applied; `CleanupContainer` nil-safe; `SessionID` stable within a process.
**Commit:** `feat: add Run entrypoint, SessionID, CleanupContainer`

---

## Phase 6 - Wait strategies (port from testcontainers-go `wait/`)

The `wait/` package depends only on `StrategyTarget` (an interface our `Container` satisfies). This is the highest-reuse phase - port near-verbatim, adapting types.

### Task 6.1: `wait/wait.go` - interfaces
**Files:**
- Create: `wait/wait.go`

`Strategy`, `StrategyTimeout`, `StrategyTarget` interfaces. `StrategyTarget` uses our `*Inspect`/`*State` and `MappedPort(ctx, port) (int, error)`. Defaults: 60s startup timeout, 100ms poll.

### Task 6.2: `wait/log.go` - ForLog
**Files:**
- Create: `wait/log.go`
- Create: `wait/log_test.go`

Port verbatim (uses `Logs`). `.AsRegexp()`, `.WithOccurrence(n)`, `.Submatch(cb)`.

### Task 6.3: `wait/port.go` - ForListeningPort/ForExposedPort/ForMappedPort
**Files:**
- Create: `wait/port.go`
- Create: `wait/port_test.go`

**Adaptation:** dial `containerIP:port` directly (no internal/external split). Uses `Host()` + `MappedPort()`.

### Task 6.4: `wait/http.go` - ForHTTP
**Files:**
- Create: `wait/http.go`
- Create: `wait/http_test.go`

Port verbatim (uses `Host`/`MappedPort` to build URL). `.WithPort`, `.WithTLS`, `.WithBasicAuth`, `.WithMethod`, `.WithStatusCodeMatcher`, `.WithResponseMatcher`.

### Task 6.5: `wait/exec.go` + `wait/exit.go` - ForExec, ForExit
**Files:**
- Create: `wait/exec.go`, `wait/exit.go`
- Create: `wait/exec_test.go`, `wait/exit_test.go`

`ForExec(cmd)` uses `StrategyTarget.Exec`. `ForExit` uses `State`.

### Task 6.6: `wait/health.go` - ForHealth
**Files:**
- Create: `wait/health.go`
- Create: `wait/health_test.go`

**Adaptation:** Apple inspect has no HEALTHCHECK field. `ForHealth()` checks `State.Status == "running"` + `ExitCode == 0` + no error. Document that users with image-defined healthchecks should use `ForExec` with the healthcheck command.

### Task 6.7: `wait/sql.go` - ForSQL
**Files:**
- Create: `wait/sql.go`
- Create: `wait/sql_test.go`

Port verbatim (uses `database/sql`, driver supplied by caller). `.WithQuery`.

### Task 6.8: `wait/file.go` - ForFile
**Files:**
- Create: `wait/file.go`
- Create: `wait/file_test.go`

Uses `CopyFileFromContainer` to check existence.

### Task 6.9: `wait/all.go` + `wait/any.go` - ForAll, ForAny + deadlines
**Files:**
- Create: `wait/all.go`, `wait/any.go`
- Create: `wait/all_test.go`, `wait/any_test.go`

Port verbatim. `ForAll` sequential, `ForAny` concurrent-first-success. `.WithDeadline(d)`.

**Tests per file:** unit tests with a fake `StrategyTarget` (recording calls, returning canned logs/inspect/state). Integration tests in Phase 12.
**Commits:** one per wait strategy file (`feat: wait ForLog`, etc.).

---

## Phase 7 - LogConsumer fan-out

### Task 7.1: `logconsumer.go` - LogConsumer + logFanout
**Files:**
- Create: `logconsumer.go`
- Create: `logconsumer_test.go`

```go
type LogConsumer interface { Accept(Log) }
type Log struct{ LogType string; Content []byte }
```

`logFanout`: starts one `container logs -f <id>` via `provider.Start`, scans lines, fans out to all registered `LogConsumer`s + exposes a `subscribe() <-chan string` for `wait.ForLog`. `Stop()` cancels the subprocess. Thread-safe registration.

**Tests:** fake subprocess reader -> assert fan-out to 2 consumers + ForLog channel.
**Commit:** `feat: add LogConsumer fan-out`

---

## Phase 8 - Network & Volume

### Task 8.1: `network.go` - Network + NetworkProvider
**Files:**
- Create: `network.go`
- Create: `network_test.go`

`Network` interface (`Remove(ctx)`, `Name()`). `NewNetwork(ctx, opts...)` wraps `container network create` (name auto-generated; `WithLabels`, `WithInternal`, `WithSubnet`, `WithSubnetV6`, `WithPlugin`). `WithNetwork(aliases, nw)`, `WithNewNetwork(ctx, aliases, opts...)` as `ContainerCustomizer`s. `CleanupNetwork(t, nw)`. Parse `network ls --format json` / `network inspect`.
**Commit:** `feat: add Network support`

### Task 8.2: `volume.go` - Volume + VolumeProvider
**Files:**
- Create: `volume.go`
- Create: `volume_test.go`

`Volume` interface (`Remove(ctx)`, `Name()`). `NewVolume(ctx, opts...)` wraps `container volume create` (`WithLabels`, `WithSize`, `WithOpt`). Parse `volume ls --format json`. Simpler than network.
**Commit:** `feat: add Volume support`

---

## Phase 9 - Build-from-file

### Task 9.1: `build.go` - FromContainerfile -> container build
**Files:**
- Create: `build.go`
- Create: `build_test.go`

`defaultBuildHook` (referenced in Task 5.1): if `req.FromContainerfile.Context != ""`, run:
```
container build -f <file> -t <tag> --progress plain [--build-arg k=v] [--target stage] [--no-cache] [--pull] [--platform p] [--secret ...] <ctx>
```
Tag = `req.FromContainerfile.Tags[0]` or generated `applecontainer-<uuid>`. Set `req.Image = tag`. `KeepImage` -> don't `image delete` on Terminate.

**Tests (fake runner):** assert exact build args; assert tag set on req; assert KeepImage skips deletion.
**Commit:** `feat: add build-from-file support`

---

## Phase 10 - Testing helpers + exec options

### Task 10.1: `testing.go` - helpers
**Files:**
- Create: `testing.go`
- Create: `testing_test.go`

`CleanupContainer` (nil-safe, swallows not-found), `CleanupNetwork`, `SkipIfProviderNotHealthy(t)` (version check + system status), `StdoutLogConsumer`. `TerminateContainer(c, opts...)` with `TerminateOptions` (`WithRemoveImage`, `WithRemoveVolumes`).
**Commit:** `feat: add testing helpers`

### Task 10.2: `exec.go` - ProcessOption for Exec
**Files:**
- Create: `exec.go`

`ProcessOption`/`ProcessOptions` (`WithUser`, `WithWorkingDir`, `WithEnv`, `Multiplexed`). Maps to `container exec` flags.
**Commit:** `feat: add exec process options`

---

## Phase 11 - Integration tests + examples

### Task 11.1: `examples/nginx_test.go`
**Files:**
- Create: `examples/nginx_test.go` (`//go:build integration`)

Run nginx, `wait.ForHTTP("/")` + `ForListeningPort("80")`, `curl` the `Endpoint`, assert 200. Tests IP mode + host-port mode.

### Task 11.2: `examples/postgres_test.go`
Run a postgres container, `wait.ForSQL("5432", "pgx", ...)` with `SELECT 1`, connect via `pgx`, assert round-trip. (pgx only in test deps.)

### Task 11.3: `examples/parallel_test.go`
`t.Parallel()` with 3 postgres containers on the same port 5432 (IP mode) - proves zero-conflict.

### Task 11.4: `examples/build_test.go`
`WithContainerfile` building a trivial image, run it, assert output.

### Task 11.5: `examples/network_test.go`
Two containers on a custom network, exec `ping <other-ip>` from one to the other.

**Run integration:** `APPLECONTAINER_INTEGRATION=1 go test -tags integration ./examples/...`
**Commits:** `test: add integration tests for nginx/postgres/parallel/build/network`

---

## Phase 12 - Polish

### Task 12.1: README
**Files:**
- Create: `README.md`

Quickstart, install (`container` CLI + macOS 26 + Apple silicon), the IP-vs-host-port explainer, wait strategies overview, integration test instructions.

### Task 12.2: `doc.go` - package docs
**Files:**
- Create: `doc.go`

### Task 12.3: Lint + vet
Run: `go vet ./...` and `gofmt -l .`
Expected: zero issues. (No external linter added to keep deps clean.)
**Commit:** `docs: add README and package docs`

---

## Verification checklist (run before declaring done)
- [ ] `go build ./...` clean
- [ ] `go vet ./...` clean
- [ ] `go test ./...` (unit, no binary needed) all green
- [ ] `APPLECONTAINER_INTEGRATION=1 go test -tags integration ./examples/...` green on macOS 26
- [ ] `container ls --all` shows no leftover test containers after a run
- [ ] README quickstart copy-paste works end-to-end

