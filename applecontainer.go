package applecontainer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/lynicis/applecontainer-go/log"
)

var (
	sessionID   string
	sessionOnce sync.Once
	versionOnce sync.Once
	versionErr  error
)

// SessionID returns a stable session identifier for the current test/run process.
func SessionID() string {
	sessionOnce.Do(func() {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		sessionID = hex.EncodeToString(b)
	})
	return sessionID
}

// versionCheckOnce runs the CLI version check exactly once.
func versionCheckOnce(ctx context.Context) error {
	versionOnce.Do(func() {
		_, versionErr = VersionCheck(ctx)
	})
	return versionErr
}

// defaultLogger resolves the logger for a container request.
func defaultLogger(_ *ContainerRequest) *slog.Logger {
	return log.Default()
}

// Run creates, starts, and waits for a container using customizers.
func Run(ctx context.Context, img string, opts ...ContainerCustomizer) (*Container, error) {
	req := &ContainerRequest{Image: img}
	for _, o := range opts {
		if err := o(req); err != nil {
			return nil, fmt.Errorf("applecontainer: customize: %w", err)
		}
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	if err := versionCheckOnce(ctx); err != nil {
		return nil, err
	}

	provider := newProvider(Read())

	// 1. Build image if requested
	if cf := req.FromContainerfile; cf.Context != "" {
		log.Printf("Building image from Containerfile in %s...", cf.Context)
		tag := ""
		if len(cf.Tags) > 0 && cf.Tags[0] != "" {
			tag = cf.Tags[0]
		} else {
			tag = randomString("applecontainer-")
		}

		args := []string{"build", "-t", tag, "--progress", "plain"}
		if cf.File != "" {
			args = append(args, "-f", cf.File)
		}
		for k, v := range cf.BuildArgs {
			if v != nil {
				args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, *v))
			}
		}
		if cf.Target != "" {
			args = append(args, "--target", cf.Target)
		}
		if cf.NoCache {
			args = append(args, "--no-cache")
		}
		if cf.Pull {
			args = append(args, "--pull")
		}
		if cf.Platform != "" {
			args = append(args, "--platform", cf.Platform)
		}
		for k, v := range cf.Secrets {
			args = append(args, "--secret", fmt.Sprintf("id=%s,src=%s", k, v))
		}
		args = append(args, cf.Context)

		if _, _, _, err := provider.runner.Run(ctx, args, nil); err != nil {
			return nil, fmt.Errorf("applecontainer: build failed: %w", err)
		}

		req.Image = tag
		if !cf.KeepImage {
			req.Cleanups = append(req.Cleanups, func(ctx context.Context) error {
				_, _, _, err := provider.runner.Run(ctx, []string{"image", "delete", tag}, nil)
				return err
			})
		}
	}

	// 2. Create container
	created, err := provider.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	c := &Container{
		provider: provider,
		id:       created.id,
		image:    req.Image,
		req:      *req,
		log:      defaultLogger(req),
	}

	// 3. Copy files to container
	for _, file := range req.Files {
		content, err := os.ReadFile(file.HostFilePath)
		if err != nil {
			return c, fmt.Errorf("applecontainer: failed to read host file %s for copy: %w", file.HostFilePath, err)
		}
		if err := c.CopyToContainer(ctx, content, file.ContainerFilePath, file.FileMode); err != nil {
			return c, fmt.Errorf("applecontainer: failed to copy file %s to container: %w", file.HostFilePath, err)
		}
	}

	// 4. Start container
	if err := c.provider.StartContainer(ctx, c); err != nil {
		return c, err
	}
	log.Printf("Container %s started.", c.GetContainerID())
	c.isRunning.Store(true)

	// 5. Setup logs if requested
	if len(req.LogWriters) > 0 {
		logCtx, cancel := context.WithCancel(context.Background())
		c.logCancel = cancel
		rc, err := c.provider.ContainerLogs(logCtx, c.id, true, 0)
		if err != nil {
			return c, fmt.Errorf("applecontainer: failed to start log follower: %w", err)
		}
		go func() {
			defer func() { _ = rc.Close() }()
			mw := io.MultiWriter(req.LogWriters...)
			_, _ = io.Copy(mw, rc)
		}()
	}

	// 6. Wait until ready
	if req.WaitingFor != nil {
		if err := req.WaitingFor.WaitUntilReady(ctx, waitTarget{c}); err != nil {
			return c, fmt.Errorf("applecontainer: wait strategy failed: %w", err)
		}
	}

	return c, nil
}

// CleanupContainer registers container termination during test cleanup.
func CleanupContainer(t testing.TB, c *Container) {
	t.Helper()
	if c == nil {
		return
	}
	t.Cleanup(func() {
		if err := c.Terminate(context.Background()); err != nil {
			t.Logf("applecontainer: failed to terminate container during cleanup: %v", err)
		}
	})
}

// Prune runs 'container prune' to clean up unused containers, networks, and volumes.
func Prune(ctx context.Context) error {
	runner := Read().runner()
	if providerRunnerOverride != nil {
		runner = providerRunnerOverride
	}
	_, _, _, err := runner.Run(ctx, []string{"prune"}, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: prune failed: %w", err)
	}
	return nil
}

// GenericLabels returns the default labels applied to all applecontainer test containers.
func GenericLabels() map[string]string {
	return map[string]string{
		"applecontainer":         "true",
		"applecontainer.session": SessionID(),
	}
}

// randomString generates a cryptographically secure random string with the given prefix.
func randomString(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
