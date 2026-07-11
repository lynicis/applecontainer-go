package benchmarks

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	applecontainer "github.com/lynicis/applecontainer-go"
)

type Runtime int

const (
	AppleContainer Runtime = iota
	TestcontainersGo
)

// RunWithBoth runs the benchmark function for both runtimes in sub-benchmarks.
func RunWithBoth(b *testing.B, fn func(b *testing.B, rt Runtime)) {
	if os.Getenv("APPLECONTAINER_BENCHMARK") == "" {
		b.Fatal("Set APPLECONTAINER_BENCHMARK=1 to run benchmarks")
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

// ImageRef resolves image name for the given runtime.
func ImageRef(name string) string {
	return name
}

// ContainerName generates a unique container name per benchmark run.
func ContainerName(b *testing.B) string {
	return fmt.Sprintf("bench-%s-%d", b.Name(), time.Now().UnixNano())
}

// prePull pulls the image outside the benchmark timer.
func prePull(t testing.TB, rt Runtime, img string) {
	t.Helper()
	switch rt {
	case AppleContainer:
		// For applecontainer, pull via the container CLI.
		_, _ = runCmd(t, "container", "image", "pull", img)
	case TestcontainersGo:
		// For testcontainers, pull via docker.
		_, _ = runCmd(t, "docker", "pull", img)
	}
}

// runCmd executes a command and returns combined output.
func runCmd(t testing.TB, name string, args ...string) (string, error) {
	t.Helper()
	/* #nosec G204 */
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
