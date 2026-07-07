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

func TestWithWaitStrategy(t *testing.T) {
	s := &dummyStrategy{name: "s"}
	req := &ContainerRequest{}

	if err := WithWaitStrategy(s)(req); err != nil {
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

	if err := WithWaitStrategyAndDeadline(s, 10*time.Second)(req); err != nil {
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

	if err := WithCLIArgsModifier(m1)(req); err != nil {
		t.Fatal(err)
	}
	if err := WithCLIArgsModifier(m2)(req); err != nil {
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
		WithLogWriters(nil),
	}

	for _, opt := range options {
		if err := opt(req); err != nil {
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
