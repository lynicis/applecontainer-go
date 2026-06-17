package applecontainer

import (
	"context"
	"testing"
	"time"

	"github.com/lynicis/applecontainer-go/wait"
)

type dummyStrategy struct {
	name string
}

func (d *dummyStrategy) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	return nil
}

func TestMergeRequest(t *testing.T) {
	s1 := &dummyStrategy{name: "s1"}
	s2 := &dummyStrategy{name: "s2"}

	var hook1Invoked, hook2Invoked bool
	h1 := func(ctx context.Context, req *ContainerRequest) error {
		hook1Invoked = true
		return nil
	}
	h2 := func(ctx context.Context, req *ContainerRequest) error {
		hook2Invoked = true
		return nil
	}

	dst := &ContainerRequest{
		Image: "nginx:latest",
		Cmd:   []string{"nginx"},
		Env:   map[string]string{"K1": "V1", "K2": "V2"},
		LifecycleHooks: ContainerLifecycleHooks{
			PreBuilds: []ContainerRequestHook{h1},
		},
	}

	src := &ContainerRequest{
		Image:             "ubuntu:latest",
		FromContainerfile: FromContainerfile{Context: "."},
		AlwaysPull:        true,
		Platform:          "linux/arm64",
		Arch:              "arm64",
		OS:                "linux",
		Cmd:               []string{"-g", "daemon off;"},
		Entrypoint:        []string{"/entrypoint.sh"},
		Env:               map[string]string{"K2": "V2-new", "K3": "V3"},
		WorkingDir:        "/app",
		User:              "appuser",
		Init:              true,
		ExposedPorts:      []string{"80/tcp"},
		HostPorts:         map[string]int{"80": 8080},
		Networks:          []string{"net1"},
		NetworkAliases:    map[string][]string{"net1": {"alias1"}},
		DNS:               []string{"8.8.8.8"},
		DNSDomain:         "local",
		DNSSearch:         []string{"search.local"},
		NoDNS:             true,
		Volumes:           []VolumeMount{{Source: "v1", Target: "/v1"}},
		Mounts:            []Mount{{Type: MountTypeBind, Source: "/src", Target: "/dst"}},
		Tmpfs:             map[string]string{"/tmp": "size=64m"},
		ShmSize:           1024,
		ReadOnlyRootfs:    true,
		Files:             []ContainerFile{{HostFilePath: "/h", ContainerFilePath: "/c"}},
		CPUs:              2.5,
		Memory:            1024 * 1024,
		CapAdd:            []string{"SYS_ADMIN"},
		CapDrop:           []string{"NET_ADMIN"},
		Ulimits:           []Ulimit{{Name: "nofile", Soft: 100, Hard: 200}},
		Rosetta:           true,
		Name:              "my-container",
		Labels:            map[string]string{"L1": "LV1"},
		WaitingFor:        s2,
		LifecycleHooks: ContainerLifecycleHooks{
			PreBuilds: []ContainerRequestHook{h2},
		},
		HostPortMapping: true,
	}

	mergeRequest(dst, src)

	// Verify scalar overrides
	if dst.Image != "ubuntu:latest" {
		t.Errorf("expected Image 'ubuntu:latest', got %q", dst.Image)
	}
	if dst.FromContainerfile.Context != "." {
		t.Errorf("expected Context '.', got %q", dst.FromContainerfile.Context)
	}
	if !dst.AlwaysPull {
		t.Error("expected AlwaysPull to be true")
	}
	if dst.Platform != "linux/arm64" {
		t.Errorf("expected Platform 'linux/arm64', got %q", dst.Platform)
	}
	if dst.Arch != "arm64" {
		t.Errorf("expected Arch 'arm64', got %q", dst.Arch)
	}
	if dst.OS != "linux" {
		t.Errorf("expected OS 'linux', got %q", dst.OS)
	}
	if dst.WorkingDir != "/app" {
		t.Errorf("expected WorkingDir '/app', got %q", dst.WorkingDir)
	}
	if dst.User != "appuser" {
		t.Errorf("expected User 'appuser', got %q", dst.User)
	}
	if !dst.Init {
		t.Error("expected Init to be true")
	}
	if dst.DNSDomain != "local" {
		t.Errorf("expected DNSDomain 'local', got %q", dst.DNSDomain)
	}
	if !dst.NoDNS {
		t.Error("expected NoDNS to be true")
	}
	if dst.ShmSize != 1024 {
		t.Errorf("expected ShmSize 1024, got %d", dst.ShmSize)
	}
	if !dst.ReadOnlyRootfs {
		t.Error("expected ReadOnlyRootfs to be true")
	}
	if dst.CPUs != 2.5 {
		t.Errorf("expected CPUs 2.5, got %f", dst.CPUs)
	}
	if dst.Memory != 1024*1024 {
		t.Errorf("expected Memory 1048576, got %d", dst.Memory)
	}
	if !dst.Rosetta {
		t.Error("expected Rosetta to be true")
	}
	if dst.Name != "my-container" {
		t.Errorf("expected Name 'my-container', got %q", dst.Name)
	}
	if dst.WaitingFor != s2 {
		t.Error("expected WaitingFor wait strategy to be src's strategy")
	}
	if !dst.HostPortMapping {
		t.Error("expected HostPortMapping to be true")
	}

	// Verify slice appends
	if len(dst.Cmd) != 3 || dst.Cmd[0] != "nginx" || dst.Cmd[1] != "-g" || dst.Cmd[2] != "daemon off;" {
		t.Errorf("unexpected Cmd: %v", dst.Cmd)
	}
	if len(dst.Entrypoint) != 1 || dst.Entrypoint[0] != "/entrypoint.sh" {
		t.Errorf("unexpected Entrypoint: %v", dst.Entrypoint)
	}
	if len(dst.ExposedPorts) != 1 || dst.ExposedPorts[0] != "80/tcp" {
		t.Errorf("unexpected ExposedPorts: %v", dst.ExposedPorts)
	}
	if len(dst.Networks) != 1 || dst.Networks[0] != "net1" {
		t.Errorf("unexpected Networks: %v", dst.Networks)
	}
	if len(dst.DNS) != 1 || dst.DNS[0] != "8.8.8.8" {
		t.Errorf("unexpected DNS: %v", dst.DNS)
	}
	if len(dst.DNSSearch) != 1 || dst.DNSSearch[0] != "search.local" {
		t.Errorf("unexpected DNSSearch: %v", dst.DNSSearch)
	}
	if len(dst.Volumes) != 1 || dst.Volumes[0].Source != "v1" {
		t.Errorf("unexpected Volumes: %v", dst.Volumes)
	}
	if len(dst.Mounts) != 1 || dst.Mounts[0].Source != "/src" {
		t.Errorf("unexpected Mounts: %v", dst.Mounts)
	}
	if len(dst.Files) != 1 || dst.Files[0].HostFilePath != "/h" {
		t.Errorf("unexpected Files: %v", dst.Files)
	}
	if len(dst.CapAdd) != 1 || dst.CapAdd[0] != "SYS_ADMIN" {
		t.Errorf("unexpected CapAdd: %v", dst.CapAdd)
	}
	if len(dst.CapDrop) != 1 || dst.CapDrop[0] != "NET_ADMIN" {
		t.Errorf("unexpected CapDrop: %v", dst.CapDrop)
	}
	if len(dst.Ulimits) != 1 || dst.Ulimits[0].Name != "nofile" {
		t.Errorf("unexpected Ulimits: %v", dst.Ulimits)
	}

	// Verify map merges
	if len(dst.Env) != 3 || dst.Env["K1"] != "V1" || dst.Env["K2"] != "V2-new" || dst.Env["K3"] != "V3" {
		t.Errorf("unexpected Env: %v", dst.Env)
	}
	if len(dst.HostPorts) != 1 || dst.HostPorts["80"] != 8080 {
		t.Errorf("unexpected HostPorts: %v", dst.HostPorts)
	}
	if len(dst.NetworkAliases) != 1 || len(dst.NetworkAliases["net1"]) != 1 || dst.NetworkAliases["net1"][0] != "alias1" {
		t.Errorf("unexpected NetworkAliases: %v", dst.NetworkAliases)
	}
	if len(dst.Tmpfs) != 1 || dst.Tmpfs["/tmp"] != "size=64m" {
		t.Errorf("unexpected Tmpfs: %v", dst.Tmpfs)
	}
	if len(dst.Labels) != 1 || dst.Labels["L1"] != "LV1" {
		t.Errorf("unexpected Labels: %v", dst.Labels)
	}

	// Verify lifecycle hook merge
	if len(dst.LifecycleHooks.PreBuilds) != 2 {
		t.Errorf("expected 2 pre-build hooks, got %d", len(dst.LifecycleHooks.PreBuilds))
	}
	_ = dst.LifecycleHooks.PreBuilds[0](context.Background(), dst)
	_ = dst.LifecycleHooks.PreBuilds[1](context.Background(), dst)
	if !hook1Invoked || !hook2Invoked {
		t.Errorf("expected both hooks to be invoked, got hook1=%v hook2=%v", hook1Invoked, hook2Invoked)
	}

	// Verify wait strategy chaining
	req := &ContainerRequest{
		WaitingFor: s1,
	}
	_ = WithAdditionalWaitStrategy(s2).Customize(req)

	composite, ok := req.WaitingFor.(*wait.ForAllStrategy)
	if !ok {
		t.Fatalf("expected composite ForAllStrategy, got %T", req.WaitingFor)
	}
	if len(composite.Strategies) != 2 || composite.Strategies[0] != s1 || composite.Strategies[1] != s2 {
		t.Errorf("expected composite with s1 and s2, got %v", composite.Strategies)
	}
	if composite.Deadline != 60*time.Second {
		t.Errorf("expected deadline 60s, got %v", composite.Deadline)
	}
}

func TestWithWaitStrategy(t *testing.T) {
	s := &dummyStrategy{name: "s"}
	req := &ContainerRequest{}

	if err := WithWaitStrategy(s).Customize(req); err != nil {
		t.Fatal(err)
	}

	composite, ok := req.WaitingFor.(*wait.ForAllStrategy)
	if !ok {
		t.Fatalf("expected ForAllStrategy, got %T", req.WaitingFor)
	}
	if len(composite.Strategies) != 1 || composite.Strategies[0] != s {
		t.Errorf("expected strategies [s], got %v", composite.Strategies)
	}
	if composite.Deadline != 60*time.Second {
		t.Errorf("expected deadline 60s, got %v", composite.Deadline)
	}
}

func TestWithWaitStrategyAndDeadline(t *testing.T) {
	s := &dummyStrategy{name: "s"}
	req := &ContainerRequest{}

	if err := WithWaitStrategyAndDeadline(s, 10*time.Second).Customize(req); err != nil {
		t.Fatal(err)
	}

	composite, ok := req.WaitingFor.(*wait.ForAllStrategy)
	if !ok {
		t.Fatalf("expected ForAllStrategy, got %T", req.WaitingFor)
	}
	if len(composite.Strategies) != 1 || composite.Strategies[0] != s {
		t.Errorf("expected strategies [s], got %v", composite.Strategies)
	}
	if composite.Deadline != 10*time.Second {
		t.Errorf("expected deadline 10s, got %v", composite.Deadline)
	}
}

func TestWithCLIArgsModifier(t *testing.T) {
	req := &ContainerRequest{}

	m1 := func(args []string) []string {
		return append(args, "m1")
	}
	m2 := func(args []string) []string {
		return append(args, "m2")
	}

	if err := WithCLIArgsModifier(m1).Customize(req); err != nil {
		t.Fatal(err)
	}
	if err := WithCLIArgsModifier(m2).Customize(req); err != nil {
		t.Fatal(err)
	}

	res := req.CLIArgsModifier([]string{"base"})
	if len(res) != 3 || res[0] != "base" || res[1] != "m1" || res[2] != "m2" {
		t.Errorf("unexpected modifier chaining: %v", res)
	}
}

func TestAllOptions(t *testing.T) {
	req := &ContainerRequest{}
	options := []ContainerCustomizer{
		WithImage("nginx:latest"),
		WithExposedPorts("80"),
		WithEnv(map[string]string{"A": "B"}),
		WithEntrypoint("/exec"),
		WithEntrypointArgs("arg1"),
		WithLabels(map[string]string{"L": "V"}),
		WithCPUs(1.5),
		WithMemory(512),
		WithCapAdd("C1"),
		WithCapDrop("C2"),
		WithUlimits(Ulimit{Name: "n", Soft: 1, Hard: 2}),
		WithWorkingDir("/dir"),
		WithUser("u"),
		WithInit(true),
		WithNetworks("n1"),
		WithNetworks("n2"),
		WithNetworkName("n3"),
		WithDNS("d1"),
		WithDNSDomain("dom"),
		WithDNSSearch("s1"),
		WithNoDNS(true),
		WithHostPortMapping(true),
		WithMounts(Mount{Source: "s", Target: "t"}),
		WithVolumes(VolumeMount{Source: "v", Target: "t"}),
		WithTmpfs(map[string]string{"t": "o"}),
		WithShmSize(100),
		WithReadOnlyRootfs(true),
		WithFiles(ContainerFile{HostFilePath: "h"}),
		WithRosetta(true),
		WithName("name"),
		WithPlatform("plat"),
		WithArch("arch"),
		WithOS("os"),
		WithAlwaysPull(true),
		WithContainerfile(FromContainerfile{Context: "c"}),
		WithLogConsumers(nil),
	}

	for _, opt := range options {
		if err := opt.Customize(req); err != nil {
			t.Fatalf("Customize failed: %v", err)
		}
	}

	// Simple assertions to ensure fields are populated
	if req.Image != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %q", req.Image)
	}
	if len(req.ExposedPorts) != 1 || req.ExposedPorts[0] != "80" {
		t.Errorf("unexpected exposed ports: %v", req.ExposedPorts)
	}
	if req.Env["A"] != "B" {
		t.Errorf("unexpected env: %v", req.Env)
	}
	if req.Entrypoint[0] != "/exec" {
		t.Errorf("unexpected entrypoint: %v", req.Entrypoint)
	}
	if req.Cmd[0] != "arg1" {
		t.Errorf("unexpected cmd: %v", req.Cmd)
	}
	if req.Labels["L"] != "V" {
		t.Errorf("unexpected labels: %v", req.Labels)
	}
	if req.CPUs != 1.5 {
		t.Errorf("unexpected cpus: %f", req.CPUs)
	}
	if req.Memory != 512 {
		t.Errorf("unexpected memory: %d", req.Memory)
	}
}
