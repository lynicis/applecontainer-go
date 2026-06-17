package applecontainer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lynicis/applecontainer-go/log"
)

// Container defines the interface for interacting with a container.
type Container interface {
	GetContainerID() string
	Endpoint(ctx context.Context, port string) (string, error)
	PortEndpoint(ctx context.Context, port string, proto string) (string, error)
	Host(context.Context) (string, error)
	MappedPort(ctx context.Context, port string) (int, error)
	ContainerIP(context.Context) (string, error)
	Inspect(context.Context) (*Inspect, error)
	State(context.Context) (*State, error)
	IsRunning() bool
	SessionID() string
	Start(context.Context) error
	Stop(context.Context, *time.Duration) error
	Terminate(ctx context.Context, opts ...TerminateOption) error
	Logs(context.Context) (io.ReadCloser, error)
	Exec(ctx context.Context, cmd []string, opts ...ProcessOption) (int, []byte, error)
	CopyToContainer(ctx context.Context, content []byte, containerPath string, mode int64) error
	CopyFileToContainer(ctx context.Context, hostPath, containerPath string, mode int64) error
	CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error)
	Networks(context.Context) ([]string, error)
}

// ContainerRequest represents the parameters for creating a container.
type ContainerRequest struct {
	Image           string
	HostPortMapping bool
}

// cliContainer implements the Container interface.
type cliContainer struct {
	provider  *cliProvider
	id        string
	image     string
	req       ContainerRequest
	log       log.Logger
	isRunning atomic.Bool
	lifecycle []ContainerLifecycleHooks
	logFanout *logFanout
}

var _ Container = (*cliContainer)(nil)

// TerminateOption is a functional option for container termination.
type TerminateOption func(*terminateOptions)

type terminateOptions struct {
	// Stub options for now
}

// ContainerLifecycleHooks defines hooks during container lifecycle phases.
type ContainerLifecycleHooks struct {
	// Stub hooks for now
}

// logFanout manages streaming container logs to multiple consumers.
type logFanout struct {
	// Stub log fanout for now
}

// GetContainerID returns the ID of the container.
func (c *cliContainer) GetContainerID() string {
	return c.id
}

// Host returns the host address of the container.
// In HostPortMapping mode, this is "localhost". Otherwise, it is the container's IP.
func (c *cliContainer) Host(ctx context.Context) (string, error) {
	if c.req.HostPortMapping {
		return "localhost", nil
	}
	return c.ContainerIP(ctx)
}

// ContainerIP returns the IP address of the container.
func (c *cliContainer) ContainerIP(ctx context.Context) (string, error) {
	ins, err := c.Inspect(ctx)
	if err != nil {
		return "", err
	}
	if len(ins.State.Networks) == 0 {
		return "", fmt.Errorf("applecontainer: no networks found for container %s", c.id)
	}
	ip := ins.State.Networks[0].IPv4()
	if ip == "" {
		return "", fmt.Errorf("applecontainer: empty IP address for container %s", c.id)
	}
	return ip, nil
}

// MappedPort returns the host-mapped port for the specified container port.
func (c *cliContainer) MappedPort(ctx context.Context, port string) (int, error) {
	pNum, proto := parsePort(port)
	if pNum <= 0 {
		return 0, fmt.Errorf("applecontainer: invalid port: %s", port)
	}
	if !c.req.HostPortMapping {
		return pNum, nil
	}
	ins, err := c.Inspect(ctx)
	if err != nil {
		return 0, err
	}
	for _, p := range ins.Configuration.PublishedPorts {
		if p.ContainerPort == pNum && (proto == "" || strings.ToLower(p.Proto) == strings.ToLower(proto)) {
			return p.HostPort, nil
		}
	}
	return 0, fmt.Errorf("applecontainer: port %s is not mapped/exposed on host", port)
}

// Endpoint returns the endpoint string (host:port) for the specified container port (defaulting to TCP).
func (c *cliContainer) Endpoint(ctx context.Context, port string) (string, error) {
	return c.PortEndpoint(ctx, port, "")
}

// PortEndpoint returns the endpoint string (host:port) for the specified container port and protocol.
func (c *cliContainer) PortEndpoint(ctx context.Context, port string, proto string) (string, error) {
	pNum, parsedProto := parsePort(port)
	if proto == "" {
		proto = parsedProto
	}
	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}
	portStr := strconv.Itoa(pNum)
	if proto != "" {
		portStr = portStr + "/" + proto
	}
	mapped, err := c.MappedPort(ctx, portStr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, mapped), nil
}

// Inspect returns the inspect metadata of the container.
func (c *cliContainer) Inspect(ctx context.Context) (*Inspect, error) {
	return c.provider.InspectContainer(ctx, c.id)
}

// State returns the state metadata of the container.
func (c *cliContainer) State(ctx context.Context) (*State, error) {
	ins, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}
	return &ins.State, nil
}

// IsRunning returns whether the container is running according to local memory state.
func (c *cliContainer) IsRunning() bool {
	return c.isRunning.Load()
}

// SessionID returns the session ID of the container.
func (c *cliContainer) SessionID() string {
	return ""
}

// Start starts the container.
func (c *cliContainer) Start(ctx context.Context) error {
	err := c.provider.StartContainer(ctx, c)
	if err != nil {
		return err
	}
	c.isRunning.Store(true)
	return nil
}

// Stop stops the container.
func (c *cliContainer) Stop(ctx context.Context, timeout *time.Duration) error {
	err := c.provider.StopContainer(ctx, c.id, timeout)
	if err != nil {
		return err
	}
	c.isRunning.Store(false)
	return nil
}

// Terminate stops and deletes the container. Terminate is idempotent.
func (c *cliContainer) Terminate(ctx context.Context, opts ...TerminateOption) error {
	c.isRunning.Store(false)

	stopErr := c.provider.StopContainer(ctx, c.id, nil)
	if stopErr != nil && !isNotFoundError(stopErr) {
		// We can log or handle stop errors, but typically we want to try deleting anyway.
	}

	delErr := c.provider.DeleteContainer(ctx, c.id, true)
	if delErr != nil && !isNotFoundError(delErr) {
		return delErr
	}

	return nil
}

// Logs returns a reader for the container's logs.
func (c *cliContainer) Logs(ctx context.Context) (io.ReadCloser, error) {
	return c.provider.ContainerLogs(ctx, c.id, false, 0)
}

// Exec executes a command inside the container.
func (c *cliContainer) Exec(ctx context.Context, cmd []string, opts ...ProcessOption) (int, []byte, error) {
	return c.provider.ExecContainer(ctx, c.id, cmd, opts...)
}

// CopyToContainer copies bytes to a path inside the container.
func (c *cliContainer) CopyToContainer(ctx context.Context, content []byte, containerPath string, mode int64) error {
	return c.provider.CopyToContainer(ctx, c.id, containerPath, content, mode)
}

// CopyFileToContainer copies a host file to a path inside the container.
func (c *cliContainer) CopyFileToContainer(ctx context.Context, hostPath, containerPath string, mode int64) error {
	content, err := os.ReadFile(hostPath)
	if err != nil {
		return fmt.Errorf("applecontainer: copy file to container: failed to read host file: %w", err)
	}
	return c.CopyToContainer(ctx, content, containerPath, mode)
}

// CopyFileFromContainer copies a file from the container.
func (c *cliContainer) CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error) {
	return c.provider.CopyFileFromContainer(ctx, c.id, path)
}

// Networks returns the names of the networks the container is attached to.
func (c *cliContainer) Networks(ctx context.Context) ([]string, error) {
	ins, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}
	var networks []string
	for _, net := range ins.State.Networks {
		networks = append(networks, net.Network)
	}
	return networks, nil
}

// Helper to parse port number and protocol from a port string (e.g. "80/tcp" or "80").
func parsePort(port string) (int, string) {
	parts := strings.Split(port, "/")
	pNum, _ := strconv.Atoi(parts[0])
	proto := ""
	if len(parts) > 1 {
		proto = parts[1]
	}
	return pNum, proto
}

// Helper to identify if an error is a "not found" or similar error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such") ||
		strings.Contains(msg, "does not exist")
}
