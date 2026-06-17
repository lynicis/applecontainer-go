//go:build integration
// +build integration

package examples

import (
	"context"
	"testing"
	"time"

	"github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
)

func TestNetworkIntegration(t *testing.T) {
	applecontainer.SkipIfProviderNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	nw, err := applecontainer.NewNetwork(ctx,
		applecontainer.WithNetworkLabels(map[string]string{"type": "integration"}),
	)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	applecontainer.CleanupNetwork(t, nw)

	c1, err := applecontainer.Run(ctx, "nginx:alpine",
		applecontainer.WithNetwork([]string{"web1"}, nw),
		applecontainer.WithWaitStrategy(wait.ForListeningPort("80")),
	)
	if err != nil {
		t.Fatalf("failed to start container 1: %v", err)
	}
	applecontainer.CleanupContainer(t, c1)

	ip1, err := c1.ContainerIP(ctx)
	if err != nil {
		t.Fatalf("failed to get c1 IP: %v", err)
	}

	c2, err := applecontainer.Run(ctx, "alpine:latest",
		applecontainer.WithNetwork([]string{"client1"}, nw),
		applecontainer.WithCmd("sleep", "60"),
		applecontainer.WithWaitStrategy(wait.ForExec([]string{"true"})),
	)
	if err != nil {
		t.Fatalf("failed to run client container: %v", err)
	}
	applecontainer.CleanupContainer(t, c2)

	exitCode, output, err := c2.Exec(ctx, []string{"ping", "-c", "3", ip1})
	if err != nil {
		t.Fatalf("failed to execute ping: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("ping failed, exit code: %d, output: %s", exitCode, string(output))
	}
}
