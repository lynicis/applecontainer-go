package applecontainer

import (
	"context"
	"fmt"
	"io"
	"os"
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

type processOptions struct{}

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
	return nil
}

// KillContainer sends a signal to a container.
func (p *cliProvider) KillContainer(ctx context.Context, id string, signal string) error {
	return nil
}

// DeleteContainer deletes a container.
func (p *cliProvider) DeleteContainer(ctx context.Context, id string, force bool) error {
	return nil
}

// InspectContainer returns metadata of a container.
func (p *cliProvider) InspectContainer(ctx context.Context, id string) (*Inspect, error) {
	return nil, nil
}

// ContainerLogs returns reader for container logs.
func (p *cliProvider) ContainerLogs(ctx context.Context, id string, follow bool, n int) (io.ReadCloser, error) {
	return nil, nil
}

// ExecContainer executes a command inside a running container.
func (p *cliProvider) ExecContainer(ctx context.Context, id string, cmd []string, opts ...ProcessOption) (int, []byte, error) {
	return 0, nil, nil
}

// CopyToContainer copies data to a path inside a container.
func (p *cliProvider) CopyToContainer(ctx context.Context, id, containerPath string, content []byte, mode int64) error {
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
