package applecontainer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	LogWriters        []io.Writer
	Cleanups          []func(context.Context) error
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
	log       *slog.Logger
	isRunning atomic.Bool
	logCancel context.CancelFunc

	waitTargetMu          sync.Mutex
	waitTargetHost        string
	waitTargetHostCached  bool
	waitTargetMappedPorts map[string]int
}

var _ Container = (*cliContainer)(nil)

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
	pNum, proto := wait.ParsePort(port)
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
	pNum, parsedProto := wait.ParsePort(port)
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
	log.Printf("Terminating container %s...", c.id)
	c.isRunning.Store(false)
	if c.logCancel != nil {
		c.logCancel()
	}

	if stopErr := c.provider.StopContainer(ctx, c.id, nil); stopErr != nil && !isNotFoundError(stopErr) {
		log.Printf("applecontainer: stop failed during terminate: %v", stopErr)
	}

	delErr := c.provider.DeleteContainer(ctx, c.id, true)
	if delErr != nil && !isNotFoundError(delErr) {
		return delErr
	}

	for _, cleanup := range c.req.Cleanups {
		if err := cleanup(ctx); err != nil {
			log.Printf("applecontainer: cleanup failed: %v", err)
		}
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
	fileMode, err := checkedFileMode(mode)
	if err != nil {
		return err
	}

	cleanHostPath := filepath.Clean(hostPath)
	info, err := os.Stat(cleanHostPath)
	if err != nil {
		return fmt.Errorf("applecontainer: copy file to container: failed to read host file: %w", err)
	}
	if info.Mode().IsRegular() && info.Mode().Perm() == fileMode.Perm() {
		return c.provider.copyHostFileToContainer(ctx, c.id, cleanHostPath, containerPath)
	}

	content, err := os.ReadFile(cleanHostPath)
	if err != nil {
		return fmt.Errorf("applecontainer: copy file to container: failed to read host file: %w", err)
	}
	return c.CopyToContainer(ctx, content, containerPath, mode)
}

// CopyFileFromContainer copies a file from the container.
func (c *cliContainer) CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error) {
	return c.provider.CopyFileFromContainer(ctx, c.id, path)
}

type waitTarget struct {
	*cliContainer
}

func normalizeWaitTargetPortKey(port string) string {
	pNum, proto := wait.ParsePort(port)
	if pNum <= 0 {
		return port
	}
	return fmt.Sprintf("%d/%s", pNum, proto)
}

func (w waitTarget) Host(ctx context.Context) (string, error) {
	w.waitTargetMu.Lock()
	if w.waitTargetHostCached {
		host := w.waitTargetHost
		w.waitTargetMu.Unlock()
		return host, nil
	}
	w.waitTargetMu.Unlock()

	host, err := w.cliContainer.Host(ctx)
	if err != nil {
		return "", err
	}

	w.waitTargetMu.Lock()
	if !w.waitTargetHostCached {
		w.waitTargetHost = host
		w.waitTargetHostCached = true
	}
	host = w.waitTargetHost
	w.waitTargetMu.Unlock()
	return host, nil
}

func (w waitTarget) MappedPort(ctx context.Context, port string) (int, error) {
	key := normalizeWaitTargetPortKey(port)

	w.waitTargetMu.Lock()
	if mapped, ok := w.waitTargetMappedPorts[key]; ok {
		w.waitTargetMu.Unlock()
		return mapped, nil
	}
	w.waitTargetMu.Unlock()

	mapped, err := w.cliContainer.MappedPort(ctx, port)
	if err != nil {
		return 0, err
	}

	w.waitTargetMu.Lock()
	if w.waitTargetMappedPorts == nil {
		w.waitTargetMappedPorts = make(map[string]int)
	}
	if cached, ok := w.waitTargetMappedPorts[key]; ok {
		w.waitTargetMu.Unlock()
		return cached, nil
	}
	w.waitTargetMappedPorts[key] = mapped
	w.waitTargetMu.Unlock()
	return mapped, nil
}

func (w waitTarget) Exec(ctx context.Context, cmd []string, opts ...any) (int, []byte, error) {
	return w.cliContainer.Exec(ctx, cmd)
}

func (w waitTarget) StateStatus(ctx context.Context) (string, error) {
	return w.cliContainer.StateStatus(ctx)
}

func (w waitTarget) StateExitCode(ctx context.Context) (int, error) {
	return w.cliContainer.StateExitCode(ctx)
}

func (w waitTarget) StateStatusAndExitCode(ctx context.Context) (string, int, error) {
	state, err := w.State(ctx)
	if err != nil {
		return "", 0, err
	}
	return state.Status, state.ExitCode, nil
}

func (w waitTarget) Logs(ctx context.Context) (io.ReadCloser, error) {
	return w.provider.ContainerLogs(ctx, w.id, true, 0)
}

func (w waitTarget) CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error) {
	return w.cliContainer.CopyFileFromContainer(ctx, path)
}
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

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such") ||
		strings.Contains(msg, "does not exist")
}
