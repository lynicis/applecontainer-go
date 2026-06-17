//go:build integration
// +build integration

package examples

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lynicis/applecontainer-go"
	"github.com/lynicis/applecontainer-go/wait"
)

func TestBuildIntegration(t *testing.T) {
	applecontainer.SkipIfProviderNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tmpDir, err := os.MkdirTemp(".", "build-context-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	containerfileContent := `FROM alpine:latest
RUN echo "hello build" > /msg.txt
CMD ["sh", "-c", "cat /msg.txt; sleep 60"]
`
	cfPath := filepath.Join(tmpDir, "Containerfile")
	err = os.WriteFile(cfPath, []byte(containerfileContent), 0644)
	if err != nil {
		t.Fatalf("failed to write Containerfile: %v", err)
	}

	c, err := applecontainer.Run(ctx, "",
		applecontainer.WithContainerfile(applecontainer.FromContainerfile{
			Context:   tmpDir,
			File:      cfPath,
			KeepImage: false,
		}),
		applecontainer.WithWaitStrategy(wait.ForLog("hello build")),
	)
	if err != nil {
		t.Fatalf("failed to build and run container: %v", err)
	}
	applecontainer.CleanupContainer(t, c)

	logsReader, err := c.Logs(ctx)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}
	defer logsReader.Close()

	logsBytes, err := io.ReadAll(logsReader)
	if err != nil {
		t.Fatalf("failed to read logs: %v", err)
	}

	if !stringsContains(string(logsBytes), "hello build") {
		t.Errorf("expected logs to contain 'hello build', got %q", string(logsBytes))
	}
}
