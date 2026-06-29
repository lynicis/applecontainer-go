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
	"github.com/lynicis/applecontainer-go/wait"
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
	Start(context.Context) error
	Stop(context.Context, *time.Duration) error
	Terminate(ctx context.Context) error
	Logs(context.Context) (io.ReadCloser, error)
	Exec(ctx context.Context, cmd []string, opts ...ProcessOption) (int, []byte, error)
	CopyToContainer(ctx context.Context, content []byte, containerPath string, mode int64) error
	CopyFileToContainer(ctx context.Context, hostPath, containerPath string, mode int64) error
	CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error)
	Networks(context.Context) ([]string, error)
}

// ContainerRequest represents the parameters for creating a container.
type ContainerRequest struct {
	Image             string
	FromContainerfile FromContainerfile
	AlwaysPull        bool
	Platform          string
	Arch              string
	OS                string
	Cmd               []string
	Entrypoint        []string
	Env               map[string]string
	WorkingDir        string
	User              string
	Init              bool
	ExposedPorts      []string
	HostPorts         map[string]int
	Networks          []string
	NetworkAliases    map[string][]string
	DNS               []string
	DNSDomain         string
	DNSSearch         []string
	NoDNS             bool
	Volumes           []VolumeMount
	Mounts            []Mount
	Tmpfs             map[string]string
	ShmSize           int64
	ReadOnlyRootfs    bool
	Files             []ContainerFile
	CPUs              float64
	Memory            int64
	CapAdd            []string
	CapDrop           []string
	Ulimits           []Ulimit
	Rosetta           bool
	Name              string
	Labels            map[string]string
	WaitingFor        WaitingFor
	LifecycleHooks    ContainerLifecycleHooks
	LogConsumerCfg    *LogConsumerCfg
	HostPortMapping   bool
	CLIArgsModifier   CLIArgsModifier
}

// FromContainerfile contains options for building an image from a Containerfile.
type FromContainerfile struct {
	Context   string             `json:"context"`
	File      string             `json:"file"`
	BuildArgs map[string]*string `json:"buildArgs"`
	Tags      []string           `json:"tags"`
	Target    string             `json:"target"`
	Platform  string             `json:"platform"`
	NoCache   bool               `json:"noCache"`
	Pull      bool               `json:"pull"`
	Secrets   map[string]string  `json:"secrets"`
	KeepImage bool               `json:"keepImage"`
}

// ContainerFile represents a file to be copied into the container.
type ContainerFile struct {
	HostFilePath      string `json:"hostFilePath"`
	ContainerFilePath string `json:"containerFilePath"`
	FileMode          int64  `json:"fileMode"`
}

// VolumeMount represents a volume mount.
type VolumeMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

// MountType defines the type of mount.
type MountType string

const (
	MountTypeBind   MountType = "bind"
	MountTypeVolume MountType = "volume"
	MountTypeTmpfs  MountType = "tmpfs"
)

// Mount represents a container mount configuration.
type Mount struct {
	Type     MountType `json:"type"`
	Source   string    `json:"source"`
	Target   string    `json:"target"`
	ReadOnly bool      `json:"readOnly"`
}

// Ulimit represents resource limit settings.
type Ulimit struct {
	Name string `json:"name"`
	Soft int64  `json:"soft"`
	Hard int64  `json:"hard"`
}

// WaitingFor is an alias for wait.Strategy.
type WaitingFor = wait.Strategy

// Log represents a log message.
type Log struct {
	LogType string
	Content []byte
}

// LogConsumer defines the interface for consuming container logs.
type LogConsumer interface {
	Accept(Log)
}

// LogConsumerCfg holds log consumer settings.
type LogConsumerCfg struct {
	Consumers []LogConsumer
}

// CLIArgsModifier allows modifying the command line arguments sent to the CLI.
type CLIArgsModifier func([]string) []string

// Validate validates the container request parameters.
func (req *ContainerRequest) Validate() error {
	hasImage := req.Image != ""
	hasFromContainerfile := req.FromContainerfile.Context != ""

	if hasImage && hasFromContainerfile {
		return fmt.Errorf("applecontainer: both Image and FromContainerfile are set, but only one is allowed")
	}
	if !hasImage && !hasFromContainerfile {
		return fmt.Errorf("applecontainer: either Image or FromContainerfile must be set")
	}

	targets := make(map[string]bool)
	for _, v := range req.Volumes {
		if v.Target == "" {
			continue
		}
		if targets[v.Target] {
			return fmt.Errorf("applecontainer: duplicate mount target: %s", v.Target)
		}
		targets[v.Target] = true
	}
	for _, m := range req.Mounts {
		if m.Target == "" {
			continue
		}
		if targets[m.Target] {
			return fmt.Errorf("applecontainer: duplicate mount target: %s", m.Target)
		}
		targets[m.Target] = true
	}

	if req.HostPortMapping && len(req.ExposedPorts) == 0 {
		return fmt.Errorf("applecontainer: HostPortMapping is enabled, but ExposedPorts is empty")
	}

	return nil
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

// ContainerRequestHook defines a hook triggered with the container request.
type ContainerRequestHook func(ctx context.Context, req *ContainerRequest) error

// ContainerHook defines a hook triggered with the container.
type ContainerHook func(ctx context.Context, c Container) error

// ContainerLifecycleHooks defines hooks during container lifecycle phases.
type ContainerLifecycleHooks struct {
	PreBuilds      []ContainerRequestHook
	PostBuilds     []ContainerRequestHook
	PreCreates     []ContainerRequestHook
	PostCreates    []ContainerRequestHook
	PreStarts      []ContainerHook
	PostStarts     []ContainerHook
	PostReadies    []ContainerHook
	PreStops       []ContainerHook
	PostStops      []ContainerHook
	PreTerminates  []ContainerHook
	PostTerminates []ContainerHook
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
	if len(ins.Networks) == 0 {
		return "", fmt.Errorf("applecontainer: no networks found for container %s", c.id)
	}
	ip := ins.Networks[0].IPv4()
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
		if p.ContainerPort == pNum && (proto == "" || strings.EqualFold(p.Proto, proto)) {
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

// StateStatus returns the container's status string.
func (c *cliContainer) StateStatus(ctx context.Context) (string, error) {
	s, err := c.State(ctx)
	if err != nil {
		return "", err
	}
	return s.Status, nil
}

// StateExitCode returns the container's exit code.
func (c *cliContainer) StateExitCode(ctx context.Context) (int, error) {
	s, err := c.State(ctx)
	if err != nil {
		return 0, err
	}
	return s.ExitCode, nil
}

// IsRunning returns whether the container is running according to local memory state.
func (c *cliContainer) IsRunning() bool {
	return c.isRunning.Load()
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
func (c *cliContainer) Terminate(ctx context.Context) error {
	c.isRunning.Store(false)

	if stopErr := c.provider.StopContainer(ctx, c.id, nil); stopErr != nil && !isNotFoundError(stopErr) {
		c.log.Printf("applecontainer: stop failed during terminate: %v", stopErr)
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
	for _, net := range ins.Networks {
		networks = append(networks, net.Network)
	}
	return networks, nil
}

func parsePort(port string) (int, string) {
	parts := strings.Split(port, "/")
	pNum, _ := strconv.Atoi(parts[0])
	proto := ""
	if len(parts) > 1 {
		proto = parts[1]
	}
	return pNum, proto
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such") ||
		strings.Contains(msg, "does not exist")
}
