# applecontainer-go

`applecontainer-go` is a lightweight, testcontainers-go-style Go library designed to spin up Apple Container (`container` CLI) Linux containers as test dependencies. It supports modern container workflows, wait strategies, lifecycle hooks, build-from-file, volumes, and custom networks.

## Prerequisites

- **Host OS**: macOS 26+ (Apple native container orchestration environment)
- **Architecture**: Apple silicon (M1/M2/M3/M4/etc.)
- **Dependency**: The Apple native `container` CLI must be installed and active (e.g. `/opt/homebrew/bin/container`).

## Quickstart

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
	ctx := context.Background()

	// Spin up Nginx in direct container IP mode
	c, err := applecontainer.Run(ctx, "nginx:alpine",
		applecontainer.WithExposedPorts("80"),
		applecontainer.WithWaitStrategy(wait.ForHTTP("/")),
	)
	if err != nil {
		panic(err)
	}
	defer c.Terminate(ctx)

	// Fetch endpoint (direct IP:80 by default)
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

## IP vs Host-Port Mode

`applecontainer-go` provides two networking models:

1. **Direct IP (Default)**: Since the Apple `container` environment maps container interfaces directly into the host bridge, you can connect directly to the container's IP (retrieved via `c.ContainerIP(ctx)` or `c.Endpoint(ctx, port)`).
2. **Host-Port Mapping (Opt-in)**: Enable by using `applecontainer.WithHostPortMapping(true)`. In this mode, the library allocates ephemeral host ports, publishes them, and resolves `c.Host(ctx)` to `"localhost"`.

## Wait Strategies

All wait strategies are ported to work with the Apple CLI seam:

- **wait.ForListeningPort**: Dials target host/port directly.
- **wait.ForHTTP**: Hits HTTP endpoints and supports status code/response body matchers.
- **wait.ForLog**: Monitors log stdout/stderr stream using regular expression or string match and occurrence count.
- **wait.ForExec**: Executes commands inside the container repeatedly until exit code (and optional response) matches.
- **wait.ForSQL**: Dials database/sql connections with query verification.
- **wait.ForFile**: Verifies file existence inside the container.
- **wait.ForHealth**: Checks running container status.
- **wait.ForAll** / **wait.ForAny**: Composite sequential or concurrent execution.

## Testing and Verification

To run unit tests (using command CLI runner fakes):

```bash
go test ./...
```

To run integration tests (spins up real containers, requires running macOS `container` CLI):

```bash
APPLECONTAINER_INTEGRATION=1 go test -tags integration -v ./examples/...
```
