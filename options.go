package applecontainer

import (
	"time"

	"github.com/lynicis/applecontainer-go/wait"
)

// ContainerCustomizer defines the interface for customizing a ContainerRequest.
type ContainerCustomizer interface {
	Customize(req *ContainerRequest) error
}

// CustomizeRequestOption is a function type that implements ContainerCustomizer.
type CustomizeRequestOption func(req *ContainerRequest) error

// Customize calls the underlying function to customize the request.
func (f CustomizeRequestOption) Customize(req *ContainerRequest) error {
	return f(req)
}

// mergeRequest copies non-empty/non-zero fields from src into dst.
// Slices are appended, maps are merged, and scalars are overridden.
func mergeRequest(dst, src *ContainerRequest) {
	if src.Image != "" {
		dst.Image = src.Image
	}
	if src.FromContainerfile.Context != "" {
		dst.FromContainerfile = src.FromContainerfile
	}
	if src.AlwaysPull {
		dst.AlwaysPull = true
	}
	if src.Platform != "" {
		dst.Platform = src.Platform
	}
	if src.Arch != "" {
		dst.Arch = src.Arch
	}
	if src.OS != "" {
		dst.OS = src.OS
	}
	if len(src.Cmd) > 0 {
		dst.Cmd = append(dst.Cmd, src.Cmd...)
	}
	if len(src.Entrypoint) > 0 {
		dst.Entrypoint = append(dst.Entrypoint, src.Entrypoint...)
	}
	if src.Env != nil {
		if dst.Env == nil {
			dst.Env = make(map[string]string)
		}
		for k, v := range src.Env {
			dst.Env[k] = v
		}
	}
	if src.WorkingDir != "" {
		dst.WorkingDir = src.WorkingDir
	}
	if src.User != "" {
		dst.User = src.User
	}
	if src.Init {
		dst.Init = true
	}
	if len(src.ExposedPorts) > 0 {
		dst.ExposedPorts = append(dst.ExposedPorts, src.ExposedPorts...)
	}
	if src.HostPorts != nil {
		if dst.HostPorts == nil {
			dst.HostPorts = make(map[string]int)
		}
		for k, v := range src.HostPorts {
			dst.HostPorts[k] = v
		}
	}
	if len(src.Networks) > 0 {
		dst.Networks = append(dst.Networks, src.Networks...)
	}
	if src.NetworkAliases != nil {
		if dst.NetworkAliases == nil {
			dst.NetworkAliases = make(map[string][]string)
		}
		for k, v := range src.NetworkAliases {
			dst.NetworkAliases[k] = append(dst.NetworkAliases[k], v...)
		}
	}
	if len(src.DNS) > 0 {
		dst.DNS = append(dst.DNS, src.DNS...)
	}
	if src.DNSDomain != "" {
		dst.DNSDomain = src.DNSDomain
	}
	if len(src.DNSSearch) > 0 {
		dst.DNSSearch = append(dst.DNSSearch, src.DNSSearch...)
	}
	if src.NoDNS {
		dst.NoDNS = true
	}
	if len(src.Volumes) > 0 {
		dst.Volumes = append(dst.Volumes, src.Volumes...)
	}
	if len(src.Mounts) > 0 {
		dst.Mounts = append(dst.Mounts, src.Mounts...)
	}
	if src.Tmpfs != nil {
		if dst.Tmpfs == nil {
			dst.Tmpfs = make(map[string]string)
		}
		for k, v := range src.Tmpfs {
			dst.Tmpfs[k] = v
		}
	}
	if src.ShmSize != 0 {
		dst.ShmSize = src.ShmSize
	}
	if src.ReadOnlyRootfs {
		dst.ReadOnlyRootfs = true
	}
	if len(src.Files) > 0 {
		dst.Files = append(dst.Files, src.Files...)
	}
	if src.CPUs != 0 {
		dst.CPUs = src.CPUs
	}
	if src.Memory != 0 {
		dst.Memory = src.Memory
	}
	if len(src.CapAdd) > 0 {
		dst.CapAdd = append(dst.CapAdd, src.CapAdd...)
	}
	if len(src.CapDrop) > 0 {
		dst.CapDrop = append(dst.CapDrop, src.CapDrop...)
	}
	if len(src.Ulimits) > 0 {
		dst.Ulimits = append(dst.Ulimits, src.Ulimits...)
	}
	if src.Rosetta {
		dst.Rosetta = true
	}
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.Labels != nil {
		if dst.Labels == nil {
			dst.Labels = make(map[string]string)
		}
		for k, v := range src.Labels {
			dst.Labels[k] = v
		}
	}
	if src.WaitingFor != nil {
		dst.WaitingFor = src.WaitingFor
	}
	// Merge lifecycle hooks
	if len(src.LifecycleHooks.PreBuilds) > 0 {
		dst.LifecycleHooks.PreBuilds = append(dst.LifecycleHooks.PreBuilds, src.LifecycleHooks.PreBuilds...)
	}
	if len(src.LifecycleHooks.PostBuilds) > 0 {
		dst.LifecycleHooks.PostBuilds = append(dst.LifecycleHooks.PostBuilds, src.LifecycleHooks.PostBuilds...)
	}
	if len(src.LifecycleHooks.PreCreates) > 0 {
		dst.LifecycleHooks.PreCreates = append(dst.LifecycleHooks.PreCreates, src.LifecycleHooks.PreCreates...)
	}
	if len(src.LifecycleHooks.PostCreates) > 0 {
		dst.LifecycleHooks.PostCreates = append(dst.LifecycleHooks.PostCreates, src.LifecycleHooks.PostCreates...)
	}
	if len(src.LifecycleHooks.PreStarts) > 0 {
		dst.LifecycleHooks.PreStarts = append(dst.LifecycleHooks.PreStarts, src.LifecycleHooks.PreStarts...)
	}
	if len(src.LifecycleHooks.PostStarts) > 0 {
		dst.LifecycleHooks.PostStarts = append(dst.LifecycleHooks.PostStarts, src.LifecycleHooks.PostStarts...)
	}
	if len(src.LifecycleHooks.PostReadies) > 0 {
		dst.LifecycleHooks.PostReadies = append(dst.LifecycleHooks.PostReadies, src.LifecycleHooks.PostReadies...)
	}
	if len(src.LifecycleHooks.PreStops) > 0 {
		dst.LifecycleHooks.PreStops = append(dst.LifecycleHooks.PreStops, src.LifecycleHooks.PreStops...)
	}
	if len(src.LifecycleHooks.PostStops) > 0 {
		dst.LifecycleHooks.PostStops = append(dst.LifecycleHooks.PostStops, src.LifecycleHooks.PostStops...)
	}
	if len(src.LifecycleHooks.PreTerminates) > 0 {
		dst.LifecycleHooks.PreTerminates = append(dst.LifecycleHooks.PreTerminates, src.LifecycleHooks.PreTerminates...)
	}
	if len(src.LifecycleHooks.PostTerminates) > 0 {
		dst.LifecycleHooks.PostTerminates = append(dst.LifecycleHooks.PostTerminates, src.LifecycleHooks.PostTerminates...)
	}
	if src.LogConsumerCfg != nil {
		if dst.LogConsumerCfg == nil {
			dst.LogConsumerCfg = &LogConsumerCfg{}
		}
		dst.LogConsumerCfg.Consumers = append(dst.LogConsumerCfg.Consumers, src.LogConsumerCfg.Consumers...)
	}
	if src.HostPortMapping {
		dst.HostPortMapping = true
	}
	if src.CLIArgsModifier != nil {
		if dst.CLIArgsModifier == nil {
			dst.CLIArgsModifier = src.CLIArgsModifier
		} else {
			oldMod := dst.CLIArgsModifier
			newMod := src.CLIArgsModifier
			dst.CLIArgsModifier = func(args []string) []string {
				return newMod(oldMod(args))
			}
		}
	}
}

// Basic options

// WithImage sets the image reference to run.
func WithImage(image string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Image = image
		return nil
	})
}

// WithExposedPorts exposes ports.
func WithExposedPorts(ports ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.ExposedPorts = append(req.ExposedPorts, ports...)
		return nil
	})
}

// WithEnv merges environment variables.
func WithEnv(env map[string]string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		if req.Env == nil {
			req.Env = make(map[string]string)
		}
		for k, v := range env {
			req.Env[k] = v
		}
		return nil
	})
}

// WithEntrypoint overrides the entrypoint.
func WithEntrypoint(entrypoint ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Entrypoint = entrypoint
		return nil
	})
}

// WithEntrypointArgs sets the command line arguments.
func WithEntrypointArgs(args ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Cmd = args
		return nil
	})
}

// WithCmd sets the command line arguments.
func WithCmd(cmd ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Cmd = cmd
		return nil
	})
}

// WithCmdArgs appends command line arguments.
func WithCmdArgs(args ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Cmd = append(req.Cmd, args...)
		return nil
	})
}

// WithLabels merges labels.
func WithLabels(labels map[string]string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		if req.Labels == nil {
			req.Labels = make(map[string]string)
		}
		for k, v := range labels {
			req.Labels[k] = v
		}
		return nil
	})
}

// WithWaitingFor sets the wait strategy.
func WithWaitingFor(s wait.Strategy) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.WaitingFor = s
		return nil
	})
}

// WithWaitStrategy sets the wait strategy, wrapping it in wait.ForAll with a 60s timeout.
func WithWaitStrategy(s wait.Strategy) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.WaitingFor = wait.ForAll(s).WithDeadline(60 * time.Second)
		return nil
	})
}

// WithWaitStrategyAndDeadline sets the wait strategy and its deadline.
func WithWaitStrategyAndDeadline(s wait.Strategy, deadline time.Duration) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.WaitingFor = wait.ForAll(s).WithDeadline(deadline)
		return nil
	})
}

// WithAdditionalWaitStrategy appends an additional wait strategy.
func WithAdditionalWaitStrategy(s wait.Strategy) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		if req.WaitingFor == nil {
			req.WaitingFor = wait.ForAll(s).WithDeadline(60 * time.Second)
		} else {
			if composite, ok := req.WaitingFor.(*wait.ForAllStrategy); ok {
				composite.Strategies = append(composite.Strategies, s)
			} else {
				req.WaitingFor = wait.ForAll(req.WaitingFor, s).WithDeadline(60 * time.Second)
			}
		}
		return nil
	})
}

// Resources options

// WithCPUs sets the number of CPUs allocated to the container.
func WithCPUs(cpus float64) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.CPUs = cpus
		return nil
	})
}

// WithMemory sets the memory limit in bytes.
func WithMemory(memory int64) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Memory = memory
		return nil
	})
}

// WithCapAdd appends capabilities to add.
func WithCapAdd(capAdd ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.CapAdd = append(req.CapAdd, capAdd...)
		return nil
	})
}

// WithCapDrop appends capabilities to drop.
func WithCapDrop(capDrop ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.CapDrop = append(req.CapDrop, capDrop...)
		return nil
	})
}

// WithUlimits appends ulimit configurations.
func WithUlimits(ulimits ...Ulimit) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Ulimits = append(req.Ulimits, ulimits...)
		return nil
	})
}

// Process options

// WithWorkingDir sets the working directory.
func WithWorkingDir(workingDir string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.WorkingDir = workingDir
		return nil
	})
}

// WithUser sets the user name or UID.
func WithUser(user string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.User = user
		return nil
	})
}

// WithInit enables or disables init mode inside container.
func WithInit(init bool) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Init = init
		return nil
	})
}

// WithEnvFile stubs loading variables from an env file.
func WithEnvFile(filePath string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		// Env file handling stub
		return nil
	})
}

// Network options

// WithNetwork appends container networks.
func WithNetwork(networks ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Networks = append(req.Networks, networks...)
		return nil
	})
}

// WithNewNetwork appends container network.
func WithNewNetwork(network string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Networks = append(req.Networks, network)
		return nil
	})
}

// WithNetworkName sets container network name.
func WithNetworkName(network string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Networks = append(req.Networks, network)
		return nil
	})
}

// WithDNS appends custom DNS servers.
func WithDNS(dns ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.DNS = append(req.DNS, dns...)
		return nil
	})
}

// WithDNSDomain sets custom DNS search domain.
func WithDNSDomain(domain string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.DNSDomain = domain
		return nil
	})
}

// WithDNSSearch appends DNS search domains.
func WithDNSSearch(search ...string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.DNSSearch = append(req.DNSSearch, search...)
		return nil
	})
}

// WithNoDNS disables auto DNS injection.
func WithNoDNS(noDNS bool) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.NoDNS = noDNS
		return nil
	})
}

// WithHostPortMapping toggles host port mapping.
func WithHostPortMapping(mapping bool) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.HostPortMapping = mapping
		return nil
	})
}

// Storage options

// WithMounts appends container mounts.
func WithMounts(mounts ...Mount) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Mounts = append(req.Mounts, mounts...)
		return nil
	})
}

// WithVolumes appends volume mounts.
func WithVolumes(volumes ...VolumeMount) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Volumes = append(req.Volumes, volumes...)
		return nil
	})
}

// WithTmpfs merges tmpfs mounts.
func WithTmpfs(tmpfs map[string]string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		if req.Tmpfs == nil {
			req.Tmpfs = make(map[string]string)
		}
		for k, v := range tmpfs {
			req.Tmpfs[k] = v
		}
		return nil
	})
}

// WithShmSize sets shared memory size.
func WithShmSize(shmSize int64) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.ShmSize = shmSize
		return nil
	})
}

// WithReadOnlyRootfs sets readonly root filesystem flag.
func WithReadOnlyRootfs(readOnly bool) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.ReadOnlyRootfs = readOnly
		return nil
	})
}

// WithFiles appends files to copy to container.
func WithFiles(files ...ContainerFile) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Files = append(req.Files, files...)
		return nil
	})
}

// Apple options

// WithRosetta enables Rosetta 2 emulation.
func WithRosetta(rosetta bool) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Rosetta = rosetta
		return nil
	})
}

// WithName sets container name.
func WithName(name string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Name = name
		return nil
	})
}

// WithPlatform sets platform.
func WithPlatform(platform string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Platform = platform
		return nil
	})
}

// WithArch sets architecture.
func WithArch(arch string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.Arch = arch
		return nil
	})
}

// WithOS sets target OS.
func WithOS(os string) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.OS = os
		return nil
	})
}

// WithAlwaysPull sets image always-pull flag.
func WithAlwaysPull(alwaysPull bool) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.AlwaysPull = alwaysPull
		return nil
	})
}

// Build options

// WithContainerfile sets options for building from a Containerfile.
func WithContainerfile(cf FromContainerfile) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		req.FromContainerfile = cf
		return nil
	})
}

// Lifecycle options

// WithLifecycleHooks appends lifecycle hooks.
func WithLifecycleHooks(hooks ContainerLifecycleHooks) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		dst := &req.LifecycleHooks
		dst.PreBuilds = append(dst.PreBuilds, hooks.PreBuilds...)
		dst.PostBuilds = append(dst.PostBuilds, hooks.PostBuilds...)
		dst.PreCreates = append(dst.PreCreates, hooks.PreCreates...)
		dst.PostCreates = append(dst.PostCreates, hooks.PostCreates...)
		dst.PreStarts = append(dst.PreStarts, hooks.PreStarts...)
		dst.PostStarts = append(dst.PostStarts, hooks.PostStarts...)
		dst.PostReadies = append(dst.PostReadies, hooks.PostReadies...)
		dst.PreStops = append(dst.PreStops, hooks.PreStops...)
		dst.PostStops = append(dst.PostStops, hooks.PostStops...)
		dst.PreTerminates = append(dst.PreTerminates, hooks.PreTerminates...)
		dst.PostTerminates = append(dst.PostTerminates, hooks.PostTerminates...)
		return nil
	})
}

// WithAdditionalLifecycleHooks is an alias to WithLifecycleHooks.
func WithAdditionalLifecycleHooks(hooks ContainerLifecycleHooks) ContainerCustomizer {
	return WithLifecycleHooks(hooks)
}

// Logging options

// WithLogConsumers registers log consumers.
func WithLogConsumers(consumers ...LogConsumer) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		if req.LogConsumerCfg == nil {
			req.LogConsumerCfg = &LogConsumerCfg{}
		}
		req.LogConsumerCfg.Consumers = append(req.LogConsumerCfg.Consumers, consumers...)
		return nil
	})
}

// WithLogger stubs logger configuration.
func WithLogger(logger any) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		return nil
	})
}

// Escape options

// WithCLIArgsModifier chains or sets custom argument modifiers.
func WithCLIArgsModifier(modifier CLIArgsModifier) ContainerCustomizer {
	return CustomizeRequestOption(func(req *ContainerRequest) error {
		if req.CLIArgsModifier == nil {
			req.CLIArgsModifier = modifier
		} else {
			oldMod := req.CLIArgsModifier
			req.CLIArgsModifier = func(args []string) []string {
				return modifier(oldMod(args))
			}
		}
		return nil
	})
}
