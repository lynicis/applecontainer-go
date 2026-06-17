package applecontainer

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

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
		pid := os.Getppid()
		now := time.Now().UnixNano()
		hash := sha1.Sum([]byte(fmt.Sprintf("applecontainer-go:%d:%d", pid, now)))
		sessionID = hex.EncodeToString(hash[:])
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
func defaultLogger(req *ContainerRequest) log.Logger {
	return log.Default()
}

// Run creates, starts, and waits for a container using customizers.
func Run(ctx context.Context, img string, opts ...ContainerCustomizer) (*cliContainer, error) {
	req := &ContainerRequest{Image: img}
	for _, o := range opts {
		if err := o.Customize(req); err != nil {
			return nil, fmt.Errorf("applecontainer: customize: %w", err)
		}
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	if err := versionCheckOnce(ctx); err != nil {
		return nil, err
	}

	provider := newCLIProvider(Read())
	c := &cliContainer{
		provider: provider,
		image:    req.Image,
		req:      *req,
		log:      defaultLogger(req),
	}

	c.lifecycle = []ContainerLifecycleHooks{
		combineContainerHooks(defaultHooks(req, c), req.LifecycleHooks),
	}

	if err := c.executeLifecycle(ctx, true); err != nil {
		return c, err
	}

	return c, nil
}

// CleanupContainer registers container termination during test cleanup.
func CleanupContainer(t testing.TB, c Container, opts ...TerminateOption) {
	t.Helper()
	if c == nil {
		return
	}
	t.Cleanup(func() {
		if err := c.Terminate(context.Background(), opts...); err != nil {
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
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return prefix + string(b)
}
