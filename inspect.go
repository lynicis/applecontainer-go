package applecontainer

// schema captured from container v1.0.0
// (`container inspect <id>` and `container list --format json` share this shape).

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Inspect is the parsed result of `container inspect <id>` and
// `container list --format json`. State is embedded so its fields
// (Networks, Status, StartedDate) are reachable directly on Inspect.
type Inspect struct {
	ID            string        `json:"id"`
	Configuration Configuration `json:"configuration"`
	State         `json:"status"`
}

// Configuration is the declared configuration of a container (the
// "configuration" object in inspect output). Named Configuration (not Config)
// to avoid colliding with the library Config singleton in config.go.
type Configuration struct {
	CapAdd           []string          `json:"capAdd"`
	CapDrop          []string          `json:"capDrop"`
	CreationDate     string            `json:"creationDate"`
	DNS              DNS               `json:"dns"`
	ID               string            `json:"id"`
	Image            Image             `json:"image"`
	InitProcess      InitProcess       `json:"initProcess"`
	Labels           map[string]string `json:"labels"`
	Mounts           []Mount           `json:"mounts"`
	Networks         []NetworkConfig   `json:"networks"`
	Platform         Platform          `json:"platform"`
	PublishedPorts   []PublishedPort   `json:"publishedPorts"`
	PublishedSockets []json.RawMessage `json:"publishedSockets"`
	ReadOnly         bool              `json:"readOnly"`
	Resources        Resources         `json:"resources"`
	Rosetta          bool              `json:"rosetta"`
	RuntimeHandler   string            `json:"runtimeHandler"`
	SSH              bool              `json:"ssh"`
	StopSignal       string            `json:"stopSignal"`
	Sysctls          map[string]string `json:"sysctls"`
	UseInit          bool              `json:"useInit"`
	Virtualization   bool              `json:"virtualization"`
}

// State is the runtime status of a container (the "status" object).
type State struct {
	Networks    []NetworkInfo `json:"networks"`
	StartedDate string        `json:"startedDate"`
	Status      string        `json:"state"`
}

// NetworkInfo is one entry under status.networks carrying the assigned IP.
type NetworkInfo struct {
	Hostname    string `json:"hostname"`
	IPv4Address string `json:"ipv4Address"`
	IPv4Gateway string `json:"ipv4Gateway"`
	IPv6Address string `json:"ipv6Address"`
	MacAddress  string `json:"macAddress"`
	MTU         int    `json:"mtu"`
	Network     string `json:"network"`
}

// IPv4 returns the IPv4 address without the CIDR suffix (e.g. "192.168.64.7"
// from "192.168.64.7/24"). If there is no "/", the value is returned as-is.
func (n NetworkInfo) IPv4() string {
	if ip, _, ok := strings.Cut(n.IPv4Address, "/"); ok {
		return ip
	}
	return n.IPv4Address
}

// DNS is the configured DNS for a container.
type DNS struct {
	Nameservers   []string `json:"nameservers"`
	Options       []string `json:"options"`
	SearchDomains []string `json:"searchDomains"`
}

// Image is the image reference plus its descriptor.
type Image struct {
	Descriptor Descriptor `json:"descriptor"`
	Reference  string     `json:"reference"`
}

// Descriptor is the OCI image descriptor.
type Descriptor struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
}

// InitProcess is the configured init process of the container.
type InitProcess struct {
	Arguments          []string          `json:"arguments"`
	Environment        []string          `json:"environment"`
	Executable         string            `json:"executable"`
	Rlimits            []json.RawMessage `json:"rlimits"`
	SupplementalGroups []int             `json:"supplementalGroups"`
	Terminal           bool              `json:"terminal"`
	User               User              `json:"user"`
	WorkingDirectory   string            `json:"workingDirectory"`
}

// User is the user specification containing the uid/gid id.
type User struct {
	ID UserID `json:"id"`
}

// UserID is the uid/gid pair identifying the container user.
type UserID struct {
	GID int `json:"gid"`
	UID int `json:"uid"`
}

// NetworkConfig is one entry under configuration.networks (the declared
// network attachment, without a runtime IP). Options is an open map because
// the keys vary (hostname, mtu, ...).
type NetworkConfig struct {
	Network string         `json:"network"`
	Options map[string]any `json:"options"`
}

// Platform is the target OS/architecture.
type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// PublishedPort is one entry under configuration.publishedPorts.
type PublishedPort struct {
	ContainerPort int    `json:"containerPort"`
	Count         int    `json:"count"`
	HostAddress   string `json:"hostAddress"`
	HostPort      int    `json:"hostPort"`
	Proto         string `json:"proto"`
}

// Resources is the resource allocation for the container.
type Resources struct {
	CPUOverhead   int   `json:"cpuOverhead"`
	CPUs          int   `json:"cpus"`
	MemoryInBytes int64 `json:"memoryInBytes"`
}

// Mount is one entry under configuration.mounts. Apple Container encodes the
// mount driver as a tagged union (e.g. {"virtiofs": {}}); the known virtiofs
// variant is captured and other drivers are retained as raw messages.
type Mount struct {
	Destination string    `json:"destination"`
	Options     []string  `json:"options"`
	Source      string    `json:"source"`
	Type        MountType `json:"type"`
}

// MountType holds the mount driver variant. VirtioFS is the only captured
// variant; its inner object is kept as a raw message so unseen fields do not
// break parsing.
type MountType struct {
	VirtioFS json.RawMessage `json:"virtiofs,omitempty"`
}

// ListEntry is an alias for Inspect: `container list --format json` returns
// the same JSON array shape as `container inspect`.
type ListEntry = Inspect

// StatsEntry is one element of `container stats --format json --no-stream`.
type StatsEntry struct {
	BlockReadBytes   int64  `json:"blockReadBytes"`
	BlockWriteBytes  int64  `json:"blockWriteBytes"`
	CPUUsageUsec     int64  `json:"cpuUsageUsec"`
	ID               string `json:"id"`
	MemoryLimitBytes int64  `json:"memoryLimitBytes"`
	MemoryUsageBytes int64  `json:"memoryUsageBytes"`
	NetworkRxBytes   int64  `json:"networkRxBytes"`
	NetworkTxBytes   int64  `json:"networkTxBytes"`
	NumProcesses     int    `json:"numProcesses"`
}

// parseInspect parses `container inspect <id>` output (a JSON array) and
// returns the first element. It requires a non-empty id. Unknown JSON fields
// are ignored.
func parseInspect(data []byte) (*Inspect, error) {
	var arr []Inspect
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("applecontainer: parse inspect: %w", err)
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("applecontainer: parse inspect: empty result")
	}
	if arr[0].ID == "" {
		return nil, fmt.Errorf("applecontainer: parse inspect: missing id")
	}
	return &arr[0], nil
}

// parseList parses `container list --format json` output (a JSON array of
// ListEntry, identical in shape to Inspect).
func parseList(data []byte) ([]ListEntry, error) {
	var arr []ListEntry
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("applecontainer: parse list: %w", err)
	}
	return arr, nil
}

// parseStats parses `container stats --format json --no-stream` output (a JSON
// array of StatsEntry).
func parseStats(data []byte) ([]StatsEntry, error) {
	var arr []StatsEntry
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("applecontainer: parse stats: %w", err)
	}
	return arr, nil
}
