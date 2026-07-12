package applecontainer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lynicis/applecontainer-go/log"
)

// ProcessOption is a functional option for container execution.
type ProcessOption func(*processOptions)

type processOptions struct {
	User       string
	WorkingDir string
	Env        []string
}

type Provider struct {
	runner commandRunner
	cfg    Config
	log    *slog.Logger
}

var providerRunnerOverride commandRunner

func newProvider(cfg Config) *Provider {
	runner := cfg.runner()
	if providerRunnerOverride != nil {
		runner = providerRunnerOverride
	}
	return &Provider{
		runner: runner,
		cfg:    cfg,
		log:    log.Default(),
	}
}

// CreateContainer creates a container but does not start it.
func (p *Provider) CreateContainer(ctx context.Context, req *ContainerRequest) (*Container, error) {
	tmpFile, err := os.CreateTemp("", "applecontainer-cid-*")
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to create cidfile: %w", err)
	}
	cidFile := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(cidFile) }()

	args, err := buildCreateArgs(req, cidFile)
	if err != nil {
		return nil, err
	}

	_, _, _, err = p.runner.Run(ctx, args, nil)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: create container failed: %w", err)
	}

	cidBytes, err := os.ReadFile(filepath.Clean(cidFile))
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to read cidfile: %w", err)
	}

	cid := strings.TrimSpace(string(cidBytes))
	if cid == "" {
		return nil, fmt.Errorf("applecontainer: cidfile is empty")
	}

	return &Container{
		provider: p,
		id:       cid,
	}, nil
}

// StartContainer starts a created container.
func (p *Provider) StartContainer(ctx context.Context, c *Container) error {
	if c == nil || c.id == "" {
		return fmt.Errorf("applecontainer: cannot start nil or empty container ID")
	}
	_, _, _, err := p.runner.Run(ctx, []string{"start", c.id}, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: start container %s failed: %w", c.id, err)
	}
	return nil
}

// StopContainer stops a running container.
func (p *Provider) StopContainer(ctx context.Context, id string, timeout *time.Duration) error {
	if id == "" {
		return fmt.Errorf("applecontainer: cannot stop empty container ID")
	}
	secs := 5
	if timeout != nil {
		secs = int(timeout.Seconds())
	}
	_, _, _, err := p.runner.Run(ctx, []string{"stop", "--time", fmt.Sprintf("%d", secs), id}, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: stop container %s failed: %w", id, err)
	}
	return nil
}

// DeleteContainer deletes a container.
func (p *Provider) DeleteContainer(ctx context.Context, id string, force bool) error {
	if id == "" {
		return fmt.Errorf("applecontainer: cannot delete empty container ID")
	}
	args := []string{"delete"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, id)
	_, _, _, err := p.runner.Run(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: delete container %s failed: %w", id, err)
	}
	return nil
}

// InspectContainer returns metadata of a container.
func (p *Provider) InspectContainer(ctx context.Context, id string) (*Inspect, error) {
	if id == "" {
		return nil, fmt.Errorf("applecontainer: cannot inspect empty container ID")
	}
	stdout, _, _, err := p.runner.Run(ctx, []string{"inspect", id}, nil)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: inspect container %s failed: %w", id, err)
	}
	ins, err := parseInspect(stdout)
	if err != nil {
		return nil, err
	}
	return ins, nil
}

type followLogsReadCloser struct {
	io.Reader
	cmd     *exec.Cmd
	onClose func()
}

func (f *followLogsReadCloser) Close() error {
	if f.onClose != nil {
		f.onClose()
	}
	if f.cmd != nil && f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
	}
	return nil
}

// ContainerLogs returns reader for container logs.
func (p *Provider) ContainerLogs(ctx context.Context, id string, follow bool, n int) (io.ReadCloser, error) {
	if id == "" {
		return nil, fmt.Errorf("applecontainer: cannot get logs for empty container ID")
	}

	args := []string{"logs"}
	if n > 0 {
		args = append(args, "-n", strconv.Itoa(n))
	}

	if follow {
		args = append(args, "-f", id)
		cmd, stdout, _, err := p.runner.Start(ctx, args, nil)
		if err != nil {
			return nil, fmt.Errorf("applecontainer: start follow logs failed: %w", err)
		}
		return &followLogsReadCloser{
			Reader: stdout,
			cmd:    cmd,
		}, nil
	}

	args = append(args, id)
	stdout, _, _, err := p.runner.Run(ctx, args, nil)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: get logs failed: %w", err)
	}

	return io.NopCloser(bytes.NewReader(stdout)), nil
}

// ExecContainer executes a command inside a running container.
func (p *Provider) ExecContainer(ctx context.Context, id string, cmd []string, opts ...ProcessOption) (int, []byte, error) {
	if id == "" {
		return -1, nil, fmt.Errorf("applecontainer: cannot exec in empty container ID")
	}
	if len(cmd) == 0 {
		return -1, nil, fmt.Errorf("applecontainer: cannot exec empty command")
	}

	var pOpts processOptions
	for _, opt := range opts {
		opt(&pOpts)
	}

	args := []string{"exec"}
	if pOpts.User != "" {
		args = append(args, "--user", pOpts.User)
	}
	if pOpts.WorkingDir != "" {
		args = append(args, "--workdir", pOpts.WorkingDir)
	}
	for _, env := range pOpts.Env {
		args = append(args, "--env", env)
	}
	args = append(args, id)
	args = append(args, cmd...)

	stdout, _, exitCode, err := p.runner.Run(ctx, args, nil)
	return exitCode, stdout, err
}

func checkedFileMode(mode int64) (os.FileMode, error) {
	if mode < 0 || mode > int64(^uint32(0)) {
		return 0, fmt.Errorf("applecontainer: invalid file mode %d", mode)
	}
	return os.FileMode(uint32(mode)), nil
}

func (p *Provider) copyHostFileToContainer(ctx context.Context, id, hostPath, containerPath string) error {
	if id == "" {
		return fmt.Errorf("applecontainer: cannot copy to empty container ID")
	}
	args := []string{"cp", filepath.Clean(hostPath), fmt.Sprintf("%s:%s", id, containerPath)}
	_, _, _, err := p.runner.Run(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: copy to container failed: %w", err)
	}
	return nil
}

// CopyToContainer copies data to a path inside a container.
func (p *Provider) CopyToContainer(ctx context.Context, id, containerPath string, content []byte, mode int64) error {
	if id == "" {
		return fmt.Errorf("applecontainer: cannot copy to empty container ID")
	}
	fileMode, err := checkedFileMode(mode)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "applecontainer-copy-*")
	if err != nil {
		return fmt.Errorf("applecontainer: failed to create temporary file for copy: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("applecontainer: failed to write content to temp file: %w", err)
	}
	_ = tmpFile.Close()

	if err := os.Chmod(tmpPath, fileMode); err != nil {
		return fmt.Errorf("applecontainer: failed to chmod temp file: %w", err)
	}

	args := []string{"cp", tmpPath, fmt.Sprintf("%s:%s", id, containerPath)}
	_, _, _, err = p.runner.Run(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: copy to container failed: %w", err)
	}
	return nil
}

type tempFileReadCloser struct {
	*os.File
}

func (t *tempFileReadCloser) Close() error {
	err := t.File.Close()
	_ = os.Remove(t.Name())
	return err
}

// CopyFileFromContainer copies a file from a container.
func (p *Provider) CopyFileFromContainer(ctx context.Context, id, path string) (io.ReadCloser, error) {
	if id == "" {
		return nil, fmt.Errorf("applecontainer: cannot copy from empty container ID")
	}
	if path == "" {
		return nil, fmt.Errorf("applecontainer: cannot copy empty path from container")
	}

	tmpFile, err := os.CreateTemp("", "applecontainer-copyfrom-*")
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to create temporary file for copy: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	args := []string{"cp", fmt.Sprintf("%s:%s", id, path), tmpPath}
	_, _, _, err = p.runner.Run(ctx, args, nil)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: copy from container failed: %w", err)
	}

	f, err := os.Open(filepath.Clean(tmpPath))
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to open copied file: %w", err)
	}

	cleanup = false
	return &tempFileReadCloser{File: f}, nil
}
