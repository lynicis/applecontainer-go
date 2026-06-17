package applecontainer

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	sessionID   string
	sessionOnce sync.Once
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

// allocateEphemeralPort allocates a free port on 127.0.0.1.
func allocateEphemeralPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	addr := l.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// parsePortNo splits a port string like "80/tcp" into (80, "tcp").
func parsePortNo(port string) (int, string) {
	parts := strings.Split(port, "/")
	pNum, _ := strconv.Atoi(parts[0])
	proto := "tcp"
	if len(parts) > 1 {
		proto = strings.ToLower(parts[1])
	}
	return pNum, proto
}

// buildCreateArgs builds the argument list for `container create` from a ContainerRequest.
func buildCreateArgs(req *ContainerRequest, cidFile string) ([]string, error) {
	args := []string{"create"}

	// Always-applied flags
	args = append(args, "--rm")
	if cidFile != "" {
		args = append(args, "--cidfile", cidFile)
	}

	// Session labels
	args = append(args, "-l", "applecontainer=true")
	args = append(args, "-l", fmt.Sprintf("applecontainer.session=%s", SessionID()))

	// Platform, Arch, OS
	if req.Platform != "" {
		args = append(args, "--platform", req.Platform)
	}
	if req.Arch != "" {
		args = append(args, "--arch", req.Arch)
	}
	if req.OS != "" {
		args = append(args, "--os", req.OS)
	}

	// Name
	if req.Name != "" {
		args = append(args, "--name", req.Name)
	}

	// Rosetta
	if req.Rosetta {
		args = append(args, "--rosetta")
	}

	// Init
	if req.Init {
		args = append(args, "--init")
	}

	// WorkingDir
	if req.WorkingDir != "" {
		args = append(args, "-w", req.WorkingDir)
	}

	// User
	if req.User != "" {
		args = append(args, "-u", req.User)
	}

	// Entrypoint
	if len(req.Entrypoint) > 0 {
		args = append(args, "--entrypoint", req.Entrypoint[0])
	}

	// CPUs
	if req.CPUs > 0 {
		args = append(args, "-c", strconv.FormatFloat(req.CPUs, 'f', -1, 64))
	}

	// Memory
	if req.Memory > 0 {
		args = append(args, "-m", strconv.FormatInt(req.Memory, 10))
	}

	// ReadOnlyRootfs
	if req.ReadOnlyRootfs {
		args = append(args, "--read-only")
	}

	// ShmSize
	if req.ShmSize > 0 {
		args = append(args, "--shm-size", strconv.FormatInt(req.ShmSize, 10))
	}

	// Env
	for k, v := range req.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Labels
	for k, v := range req.Labels {
		args = append(args, "-l", fmt.Sprintf("%s=%s", k, v))
	}

	// CapAdd
	for _, cap := range req.CapAdd {
		args = append(args, "--cap-add", cap)
	}

	// CapDrop
	for _, cap := range req.CapDrop {
		args = append(args, "--cap-drop", cap)
	}

	// Networks
	for _, network := range req.Networks {
		args = append(args, "--network", network)
	}

	// Port publishing
	if len(req.ExposedPorts) > 0 {
		if req.HostPorts == nil {
			req.HostPorts = make(map[string]int)
		}
		for _, portStr := range req.ExposedPorts {
			containerPort, proto := parsePortNo(portStr)
			if containerPort <= 0 {
				continue
			}

			if req.HostPortMapping {
				hostPort, ok := req.HostPorts[portStr]
				if !ok || hostPort <= 0 {
					hp, err := allocateEphemeralPort()
					if err != nil {
						return nil, fmt.Errorf("applecontainer: failed to allocate ephemeral port for %s: %w", portStr, err)
					}
					hostPort = hp
					req.HostPorts[portStr] = hostPort
				}
				args = append(args, "--publish", fmt.Sprintf("%d:%d/%s", hostPort, containerPort, proto))
			}
		}
	}

	// Volumes
	for _, v := range req.Volumes {
		opt := ""
		if v.ReadOnly {
			opt = ":ro"
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s%s", v.Source, v.Target, opt))
	}

	// Mounts
	for _, m := range req.Mounts {
		opt := ""
		if m.ReadOnly {
			opt = ",readonly"
		}
		args = append(args, "--mount", fmt.Sprintf("type=%s,source=%s,target=%s%s", m.Type, m.Source, m.Target, opt))
	}

	// Tmpfs
	for path, opts := range req.Tmpfs {
		val := path
		if opts != "" {
			val = fmt.Sprintf("%s:%s", path, opts)
		}
		args = append(args, "--tmpfs", val)
	}

	// Positional arguments: Image followed by Cmd
	args = append(args, req.Image)
	if len(req.Cmd) > 0 {
		args = append(args, req.Cmd...)
	}

	// CLIArgsModifier applied last
	if req.CLIArgsModifier != nil {
		args = req.CLIArgsModifier(args)
	}

	return args, nil
}
