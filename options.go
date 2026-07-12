package applecontainer

import (
	"io"
	"time"

	"github.com/lynicis/applecontainer-go/wait"
)

// ContainerCustomizer is a function that customizes a ContainerRequest.
type ContainerCustomizer func(req *ContainerRequest) error

// WithImage sets the image reference to run.
func WithImage(image string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Image = image
		return nil
	}
}

// WithExposedPorts exposes ports.
func WithExposedPorts(ports ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.ExposedPorts = append(req.ExposedPorts, ports...)
		return nil
	}
}

// WithEnv merges environment variables.
func WithEnv(env map[string]string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		if req.Env == nil {
			req.Env = make(map[string]string)
		}
		for k, v := range env {
			req.Env[k] = v
		}
		return nil
	}
}

// WithEntrypoint overrides the entrypoint.
func WithEntrypoint(entrypoint ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Entrypoint = entrypoint
		return nil
	}
}

// WithEntrypointArgs sets the command line arguments.
func WithEntrypointArgs(args ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Cmd = args
		return nil
	}
}

// WithCmd sets the command line arguments.
func WithCmd(cmd ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Cmd = cmd
		return nil
	}
}

// WithCmdArgs appends command line arguments.
func WithCmdArgs(args ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Cmd = append(req.Cmd, args...)
		return nil
	}
}

// WithLabels merges labels.
func WithLabels(labels map[string]string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		if req.Labels == nil {
			req.Labels = make(map[string]string)
		}
		for k, v := range labels {
			req.Labels[k] = v
		}
		return nil
	}
}

// WithWaitingFor sets the wait strategy.
func WithWaitingFor(s wait.Strategy) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.WaitingFor = s
		return nil
	}
}

// WithWaitStrategy sets the wait strategy, wrapping it in wait.ForAll with a 60s timeout.
func WithWaitStrategy(s wait.Strategy) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.WaitingFor = wait.ForAll(s).WithDeadline(60 * time.Second)
		return nil
	}
}

// WithCPUs sets the number of CPUs allocated to the container.
func WithCPUs(cpus float64) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.CPUs = cpus
		return nil
	}
}

// WithMemory sets the memory limit in bytes.
func WithMemory(memory int64) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Memory = memory
		return nil
	}
}

// WithCapAdd appends capabilities to add.
func WithCapAdd(capAdd ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.CapAdd = append(req.CapAdd, capAdd...)
		return nil
	}
}

// WithCapDrop appends capabilities to drop.
func WithCapDrop(capDrop ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.CapDrop = append(req.CapDrop, capDrop...)
		return nil
	}
}

// WithUlimits appends ulimit configurations.
func WithUlimits(ulimits ...Ulimit) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Ulimits = append(req.Ulimits, ulimits...)
		return nil
	}
}

// WithWorkingDir sets the working directory.
func WithWorkingDir(workingDir string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.WorkingDir = workingDir
		return nil
	}
}

// WithUser sets the user name or UID.
func WithUser(user string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.User = user
		return nil
	}
}

// WithInit enables or disables init mode inside container.
func WithInit(init bool) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Init = init
		return nil
	}
}

// WithNetworks appends container networks.
func WithNetworks(networks ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Networks = append(req.Networks, networks...)
		return nil
	}
}

// WithNetworkName sets container network name.
func WithNetworkName(network string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Networks = append(req.Networks, network)
		return nil
	}
}

// WithDNS appends custom DNS servers.
func WithDNS(dns ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.DNS = append(req.DNS, dns...)
		return nil
	}
}

// WithDNSDomain sets custom DNS search domain.
func WithDNSDomain(domain string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.DNSDomain = domain
		return nil
	}
}

// WithDNSSearch appends DNS search domains.
func WithDNSSearch(search ...string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.DNSSearch = append(req.DNSSearch, search...)
		return nil
	}
}

// WithNoDNS disables auto DNS injection.
func WithNoDNS(noDNS bool) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.NoDNS = noDNS
		return nil
	}
}

// WithHostPortMapping toggles host port mapping.
func WithHostPortMapping(mapping bool) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.HostPortMapping = mapping
		return nil
	}
}

// WithMounts appends container mounts.
func WithMounts(mounts ...Mount) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Mounts = append(req.Mounts, mounts...)
		return nil
	}
}

// WithVolumes appends volume mounts.
func WithVolumes(volumes ...VolumeMount) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Volumes = append(req.Volumes, volumes...)
		return nil
	}
}

// WithTmpfs merges tmpfs mounts.
func WithTmpfs(tmpfs map[string]string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		if req.Tmpfs == nil {
			req.Tmpfs = make(map[string]string)
		}
		for k, v := range tmpfs {
			req.Tmpfs[k] = v
		}
		return nil
	}
}

// WithShmSize sets shared memory size.
func WithShmSize(shmSize int64) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.ShmSize = shmSize
		return nil
	}
}

// WithReadOnlyRootfs sets readonly root filesystem flag.
func WithReadOnlyRootfs(readOnly bool) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.ReadOnlyRootfs = readOnly
		return nil
	}
}

// WithFiles appends files to copy to container.
func WithFiles(files ...ContainerFile) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Files = append(req.Files, files...)
		return nil
	}
}

// WithRosetta enables Rosetta 2 emulation.
func WithRosetta(rosetta bool) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Rosetta = rosetta
		return nil
	}
}

// WithName sets container name.
func WithName(name string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Name = name
		return nil
	}
}

// WithPlatform sets platform.
func WithPlatform(platform string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Platform = platform
		return nil
	}
}

// WithArch sets architecture.
func WithArch(arch string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Arch = arch
		return nil
	}
}

// WithOS sets target OS.
func WithOS(os string) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.OS = os
		return nil
	}
}

// WithAlwaysPull sets image always-pull flag.
func WithAlwaysPull(alwaysPull bool) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.AlwaysPull = alwaysPull
		return nil
	}
}

// WithContainerfile sets options for building from a Containerfile.
func WithContainerfile(cf FromContainerfile) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.FromContainerfile = cf
		return nil
	}
}

// WithLogWriters registers log writers.
func WithLogWriters(writers ...io.Writer) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.LogWriters = append(req.LogWriters, writers...)
		return nil
	}
}
