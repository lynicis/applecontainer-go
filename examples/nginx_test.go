//go:build integration
// +build integration

package examples

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
)

func TestNginxIntegration(t *testing.T) {
	applecontainer.SkipIfProviderNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Test in default direct IP mode
	t.Run("DirectIP", func(t *testing.T) {
		c, err := applecontainer.Run(ctx, "nginx:alpine",
			applecontainer.WithExposedPorts("80"),
			applecontainer.WithWaitStrategy(wait.ForHTTP("/").WithPort("80")),
		)
		if err != nil {
			t.Fatalf("failed to start nginx in direct IP mode: %v", err)
		}
		applecontainer.CleanupContainer(t, c)

		endpoint, err := c.Endpoint(ctx, "80")
		if err != nil {
			t.Fatalf("failed to get endpoint: %v", err)
		}

		resp, err := http.Get("http://" + endpoint)
		if err != nil {
			t.Fatalf("failed to GET nginx: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	// 2. Test in host-port mapping mode
	t.Run("HostPortMapping", func(t *testing.T) {
		c, err := applecontainer.Run(ctx, "nginx:alpine",
			applecontainer.WithExposedPorts("80"),
			applecontainer.WithHostPortMapping(true),
			applecontainer.WithWaitStrategy(wait.ForHTTP("/").WithPort("80")),
		)
		if err != nil {
			t.Fatalf("failed to start nginx in host-port mapping mode: %v", err)
		}
		applecontainer.CleanupContainer(t, c)

		endpoint, err := c.Endpoint(ctx, "80")
		if err != nil {
			t.Fatalf("failed to get endpoint: %v", err)
		}

		resp, err := http.Get("http://" + endpoint)
		if err != nil {
			t.Fatalf("failed to GET nginx: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !stringsContains(string(body), "Welcome to nginx!") {
			t.Errorf("unexpected body content: %s", string(body))
		}
	})
}

func stringsContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
