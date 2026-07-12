package applecontainer

import (
	"fmt"
	"net"
	"strconv"

	"github.com/lynicis/applecontainer-go/wait"
)

// allocateEphemeralPort allocates a free port on 127.0.0.1.
func allocateEphemeralPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	addr := l.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// wait.ParsePort splits a port string like "80/tcp" into (80, "tcp").

// buildCreateArgs builds the argument list for `container create` from a ContainerRequest.
func buildCreateArgs(req *ContainerRequest, cidFile string) ([]string, error) {
	args := []string{"create"}

	args = append(args, "--rm")
	if cidFile != "" {
		args = append(args, "--cidfile", cidFile)
	}

	args = append(args, "-l", "applecontainer=true")
	args = append(args, "-l", fmt.Sprintf("applecontainer.session=%s", SessionID()))

	if req.Platform != "" {
		args = append(args, "--platform", req.Platform)
	}
	if req.Arch != "" {
		args = append(args, "--arch", req.Arch)
	}
	if req.OS != "" {
		args = append(args, "--os", req.OS)
	}

	if req.Name != "" {
		args = append(args, "--name", req.Name)
	}

	if req.Rosetta {
		args = append(args, "--rosetta")
	}

	if req.Init {
		args = append(args, "--init")
	}

	if req.WorkingDir != "" {
		args = append(args, "-w", req.WorkingDir)
	}

	if req.User != "" {
		args = append(args, "-u", req.User)
	}

	if len(req.Entrypoint) > 0 {
		args = append(args, "--entrypoint", req.Entrypoint[0])
	}

	if req.CPUs > 0 {
		args = append(args, "-c", strconv.FormatFloat(req.CPUs, 'f', -1, 64))
	}

	if req.Memory > 0 {
		args = append(args, "-m", strconv.FormatInt(req.Memory, 10))
	}

	if req.ReadOnlyRootfs {
		args = append(args, "--read-only")
	}

	if req.ShmSize > 0 {
		args = append(args, "--shm-size", strconv.FormatInt(req.ShmSize, 10))
	}

	for k, v := range req.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range req.Labels {
		args = append(args, "-l", fmt.Sprintf("%s=%s", k, v))
	}

	for _, cap := range req.CapAdd {
		args = append(args, "--cap-add", cap)
	}

	for _, cap := range req.CapDrop {
		args = append(args, "--cap-drop", cap)
	}

	for _, network := range req.Networks {
		args = append(args, "--network", network)
	}

	if len(req.ExposedPorts) > 0 {
		if req.HostPorts == nil {
			req.HostPorts = make(map[string]int)
		}
		for _, portStr := range req.ExposedPorts {
			containerPort, proto := wait.ParsePort(portStr)
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

	for _, v := range req.Volumes {
		opt := ""
		if v.ReadOnly {
			opt = ":ro"
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s%s", v.Source, v.Target, opt))
	}

	for _, m := range req.Mounts {
		opt := ""
		if m.ReadOnly {
			opt = ",readonly"
		}
		args = append(args, "--mount", fmt.Sprintf("type=%s,source=%s,target=%s%s", m.Type, m.Source, m.Target, opt))
	}

	for path, opts := range req.Tmpfs {
		val := path
		if opts != "" {
			val = fmt.Sprintf("%s:%s", path, opts)
		}
		args = append(args, "--tmpfs", val)
	}

	args = append(args, req.Image)
	if len(req.Cmd) > 0 {
		args = append(args, req.Cmd...)
	}
	return args, nil
}
