package applecontainer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/lynicis/applecontainer-go/log"
)

var (
	sessionID   string
	sessionOnce sync.Once
	versionMu   sync.Mutex
	versionOK   bool
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

// ensureVersion runs the CLI version check until it succeeds; failures are retried.
func ensureVersion(ctx context.Context) error {
	versionMu.Lock()
	defer versionMu.Unlock()
	if versionOK {
		return nil
	}
	if _, err := VersionCheck(ctx); err != nil {
		return err
	}
	versionOK = true
	return nil
}

// resetVersionCheck clears the cached version-check result.
func resetVersionCheck() {
	versionMu.Lock()
	versionOK = false
	versionMu.Unlock()
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

	if err := ensureVersion(ctx); err != nil {
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

	// 2. Pull image when requested (a Containerfile build already refreshes it).
	if req.AlwaysPull && req.FromContainerfile.Context == "" {
		if _, _, _, err := provider.runner.Run(ctx, buildPullArgs(req), nil); err != nil {
			return nil, fmt.Errorf("applecontainer: pull image %s: %w", req.Image, err)
		}
	}

	// 3. Create container
	created, err := provider.CreateContainer(ctx, req)
	if err != nil {
		runCleanups(context.WithoutCancel(ctx), req.Cleanups)
		return nil, err
	}

	c := &Container{
		provider: provider,
		id:       created.id,
		req:      *req,
	}
	keep := false
	defer func() {
		if !keep {
			if err := c.Terminate(context.WithoutCancel(ctx)); err != nil {
				log.Printf("applecontainer: cleanup failed: %v", err)
			}
		}
	}()

	// 4. Copy files to container
	for _, file := range req.Files {
		if err := c.CopyFileToContainer(ctx, file.HostFilePath, file.ContainerFilePath, file.FileMode); err != nil {
			return nil, fmt.Errorf("applecontainer: failed to copy file %s to container: %w", file.HostFilePath, err)
		}
	}

	// 5. Start container
	if err := c.provider.StartContainer(ctx, c); err != nil {
		return nil, err
	}
	log.Printf("Container %s started.", c.GetContainerID())
	c.isRunning.Store(true)

	// 6. Setup logs if requested
	if len(req.LogWriters) > 0 {
		logCtx, cancel := context.WithCancel(context.Background())
		c.logCancel = cancel
		rc, err := c.provider.ContainerLogs(logCtx, c.id, true, 0)
		if err != nil {
			return nil, fmt.Errorf("applecontainer: failed to start log follower: %w", err)
		}
		go func() {
			defer func() { _ = rc.Close() }()
			mw := io.MultiWriter(req.LogWriters...)
			_, _ = io.Copy(mw, rc)
		}()
	}

	// 7. Wait until ready
	if req.WaitingFor != nil {
		if err := req.WaitingFor.WaitUntilReady(ctx, waitTarget{c}); err != nil {
			return nil, fmt.Errorf("applecontainer: wait strategy failed: %w", err)
		}
	}

	keep = true
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
