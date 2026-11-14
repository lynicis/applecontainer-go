package benchmarks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
)

const (
	imageAlpine   = "alpine:latest"
	imageNginx    = "nginx:alpine"
	imagePostgres = "postgres:15-alpine"
	imageRedis    = "redis:alpine"
)

type Runtime int

const (
	AppleContainer Runtime = iota
	TestcontainersGo
)

// RunWithBoth runs the benchmark function for both runtimes in sub-benchmarks.
func RunWithBoth(b *testing.B, fn func(b *testing.B, rt Runtime)) {
	if os.Getenv("APPLECONTAINER_BENCHMARK") == "" {
		b.Skip("Set APPLECONTAINER_BENCHMARK=1 to run benchmarks")
	}

	b.Run("applecontainer", func(b *testing.B) {
		b.ReportAllocs()
		SkipIfProviderNotHealthy(b)
		fn(b, AppleContainer)
	})
	b.Run("testcontainers-go", func(b *testing.B) {
		b.ReportAllocs()
		SkipIfDockerNotHealthy(b)
		fn(b, TestcontainersGo)
	})
}

// SkipIfDockerNotHealthy skips if Docker is unavailable.
func SkipIfDockerNotHealthy(t testing.TB) {
	t.Helper()
	if _, err := runCmd(t, "docker", "info"); err != nil {
		t.Skip("Docker is not healthy, skipping")
	}
}

// SkipIfProviderNotHealthy skips if Apple container runtime is unavailable.
func SkipIfProviderNotHealthy(t testing.TB) {
	t.Helper()
	applecontainer.SkipIfProviderNotHealthy(t)
}

var pulled sync.Map

// prePull pulls the image outside the benchmark timer. Each runtime/image pair
// is pulled at most once per process; a failed pull fails the benchmark.
func prePull(t testing.TB, rt Runtime, img string) {
	t.Helper()
	key := fmt.Sprintf("%d/%s", rt, img)
	if _, ok := pulled.Load(key); ok {
		return
	}
	var out string
	var err error
	switch rt {
	case AppleContainer:
		out, err = runCmd(t, "container", "image", "pull", img)
	case TestcontainersGo:
		out, err = runCmd(t, "docker", "pull", img)
	}
	if err != nil {
		t.Fatalf("pull %s: %v\n%s", img, err, out)
	}
	pulled.Store(key, struct{}{})
}

// runCmd executes a command and returns combined output.
func runCmd(t testing.TB, name string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	/* #nosec G204 */
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
