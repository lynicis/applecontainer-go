package applecontainer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lynicis/applecontainer-go/log"
)

// ContainerRequest represents the parameters for creating a container.
type ContainerRequest struct {
	Image string
}

// cliContainer is the library's implementation of the Container interface.
type cliContainer struct {
	provider *cliProvider
	id       string
}

// ProcessOption is a functional option for container execution.
type ProcessOption func(*processOptions)

type processOptions struct {
	User       string
	WorkingDir string
	Env        []string
}

// PullOption is a functional option for image pulling.
type PullOption func(*pullOptions)

type pullOptions struct{}

// ImageInspect represents metadata of an image.
type ImageInspect struct {
	ID string
}

// ContainerProvider defines the interface for managing container lifecycles.
type ContainerProvider interface {
	CreateContainer(ctx context.Context, req *ContainerRequest) (*cliContainer, error)
	StartContainer(ctx context.Context, c *cliContainer) error
	StopContainer(ctx context.Context, id string, timeout *time.Duration) error
	KillContainer(ctx context.Context, id string, signal string) error
	DeleteContainer(ctx context.Context, id string, force bool) error
	InspectContainer(ctx context.Context, id string) (*Inspect, error)
	ContainerLogs(ctx context.Context, id string, follow bool, n int) (io.ReadCloser, error)
	ExecContainer(ctx context.Context, id string, cmd []string, opts ...ProcessOption) (int, []byte, error)
	CopyToContainer(ctx context.Context, id, containerPath string, content []byte, mode int64) error
	CopyFileFromContainer(ctx context.Context, id, path string) (io.ReadCloser, error)
	ImagePull(ctx context.Context, ref string, opts ...PullOption) error
	ImageInspect(ctx context.Context, ref string) (*ImageInspect, error)
	Health(ctx context.Context) error
	Close() error
}

type cliProvider struct {
	runner commandRunner
	cfg    Config
	log    log.Logger
}

var _ ContainerProvider = (*cliProvider)(nil)

func newCLIProvider(cfg Config) *cliProvider {
	return &cliProvider{
		runner: cfg.runner(),
		cfg:    cfg,
		log:    log.Default(),
	}
}

func buildCreateArgs(req *ContainerRequest, cidFile string) []string {
	return []string{"create", "--rm", "--cidfile", cidFile, req.Image}
}

// CreateContainer creates a container but does not start it.
func (p *cliProvider) CreateContainer(ctx context.Context, req *ContainerRequest) (*cliContainer, error) {
	tmpFile, err := os.CreateTemp("", "applecontainer-cid-*")
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to create cidfile: %w", err)
	}
	cidFile := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(cidFile)

	args := buildCreateArgs(req, cidFile)

	_, _, _, err = p.runner.Run(ctx, args, nil)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: create container failed: %w", err)
	}

	cidBytes, err := os.ReadFile(cidFile)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to read cidfile: %w", err)
	}

	cid := strings.TrimSpace(string(cidBytes))
	if cid == "" {
		return nil, fmt.Errorf("applecontainer: cidfile is empty")
	}

	return &cliContainer{
		provider: p,
		id:       cid,
	}, nil
}

// StartContainer starts a created container.
func (p *cliProvider) StartContainer(ctx context.Context, c *cliContainer) error {
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
func (p *cliProvider) StopContainer(ctx context.Context, id string, timeout *time.Duration) error {
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

// KillContainer sends a signal to a container.
func (p *cliProvider) KillContainer(ctx context.Context, id string, signal string) error {
	if id == "" {
		return fmt.Errorf("applecontainer: cannot kill empty container ID")
	}
	args := []string{"kill"}
	if signal != "" {
		args = append(args, "--signal", signal)
	}
	args = append(args, id)
	_, _, _, err := p.runner.Run(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: kill container %s failed: %w", id, err)
	}
	return nil
}

// DeleteContainer deletes a container.
func (p *cliProvider) DeleteContainer(ctx context.Context, id string, force bool) error {
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
func (p *cliProvider) InspectContainer(ctx context.Context, id string) (*Inspect, error) {
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
func (p *cliProvider) ContainerLogs(ctx context.Context, id string, follow bool, n int) (io.ReadCloser, error) {
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
func (p *cliProvider) ExecContainer(ctx context.Context, id string, cmd []string, opts ...ProcessOption) (int, []byte, error) {
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

// CopyToContainer copies data to a path inside a container.
func (p *cliProvider) CopyToContainer(ctx context.Context, id, containerPath string, content []byte, mode int64) error {
	if id == "" {
		return fmt.Errorf("applecontainer: cannot copy to empty container ID")
	}
	tmpFile, err := os.CreateTemp("", "applecontainer-copy-*")
	if err != nil {
		return fmt.Errorf("applecontainer: failed to create temporary file for copy: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("applecontainer: failed to write content to temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, os.FileMode(mode)); err != nil {
		return fmt.Errorf("applecontainer: failed to chmod temp file: %w", err)
	}

	args := []string{"cp", tmpPath, fmt.Sprintf("%s:%s", id, containerPath)}
	_, _, _, err = p.runner.Run(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: copy to container failed: %w", err)
	}
	return nil
}

// CopyFileFromContainer copies a file from a container.
func (p *cliProvider) CopyFileFromContainer(ctx context.Context, id, path string) (io.ReadCloser, error) {
	return nil, nil
}

// ImagePull pulls an image from a registry.
func (p *cliProvider) ImagePull(ctx context.Context, ref string, opts ...PullOption) error {
	return nil
}

// ImageInspect returns metadata of an image.
func (p *cliProvider) ImageInspect(ctx context.Context, ref string) (*ImageInspect, error) {
	return nil, nil
}

// Health checks the health of the container provider.
func (p *cliProvider) Health(ctx context.Context) error {
	return nil
}

// Close closes any resources associated with the provider.
func (p *cliProvider) Close() error {
	return nil
}
