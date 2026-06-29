package applecontainer

import (
	"os"
	"testing"
)

func TestParseInspectRoundTrip(t *testing.T) {
	data, err := os.ReadFile("testdata/inspect.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseInspect(data)
	if err != nil {
		t.Fatalf("parseInspect: %v", err)
	}
	if got.ID == "" {
		t.Fatal("empty ID")
	}
	if len(got.Networks) == 0 {
		t.Fatal("no networks")
	}
	if got.Networks[0].IPv4Address == "" {
		t.Fatal("empty ipv4")
	}
}

func TestParseInspectEmbeddedState(t *testing.T) {
	data, err := os.ReadFile("testdata/inspect.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseInspect(data)
	if err != nil {
		t.Fatalf("parseInspect: %v", err)
	}
	if got.Status != "running" {
		t.Fatalf("Status=%q want running", got.Status)
	}
	if got.StartedDate == "" {
		t.Fatal("empty StartedDate")
	}
	if got.State.Networks[0].Network != "default" {
		t.Fatalf("network=%q want default", got.State.Networks[0].Network)
	}
}

func TestParseInspectConfiguration(t *testing.T) {
	data, err := os.ReadFile("testdata/inspect.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseInspect(data)
	if err != nil {
		t.Fatalf("parseInspect: %v", err)
	}
	if got.Configuration.Image.Reference == "" {
		t.Fatal("empty image reference")
	}
	if got.Configuration.Platform.Architecture != "arm64" {
		t.Fatalf("arch=%q want arm64", got.Configuration.Platform.Architecture)
	}
	if got.Configuration.Platform.OS != "linux" {
		t.Fatalf("os=%q want linux", got.Configuration.Platform.OS)
	}
	if got.Configuration.Resources.CPUs == 0 {
		t.Fatal("zero CPUs")
	}
	if got.Configuration.Resources.MemoryInBytes == 0 {
		t.Fatal("zero memory")
	}
	if got.Configuration.Rosetta != false {
		t.Fatal("rosetta want false")
	}
	if got.Configuration.RuntimeHandler == "" {
		t.Fatal("empty runtime handler")
	}
	if len(got.Configuration.InitProcess.Environment) == 0 {
		t.Fatal("no init process env")
	}
	if got.Configuration.InitProcess.Executable == "" {
		t.Fatal("empty init process executable")
	}
}

func TestParseInspectPublishedPort(t *testing.T) {
	data, err := os.ReadFile("testdata/inspect-published-port.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseInspect(data)
	if err != nil {
		t.Fatalf("parseInspect: %v", err)
	}
	ports := got.Configuration.PublishedPorts
	if len(ports) != 1 {
		t.Fatalf("published ports=%d want 1", len(ports))
	}
	p := ports[0]
	if p.HostPort != 8080 {
		t.Fatalf("hostPort=%d want 8080", p.HostPort)
	}
	if p.ContainerPort != 80 {
		t.Fatalf("containerPort=%d want 80", p.ContainerPort)
	}
	if p.Proto != "tcp" {
		t.Fatalf("proto=%q want tcp", p.Proto)
	}
	if p.HostAddress != "0.0.0.0" {
		t.Fatalf("hostAddress=%q want 0.0.0.0", p.HostAddress)
	}
}

func TestParseInspectMount(t *testing.T) {
	data, err := os.ReadFile("testdata/inspect-mount.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseInspect(data)
	if err != nil {
		t.Fatalf("parseInspect: %v", err)
	}
	mounts := got.Configuration.Mounts
	if len(mounts) != 1 {
		t.Fatalf("mounts=%d want 1", len(mounts))
	}
	m := mounts[0]
	if m.Source != "/tmp" {
		t.Fatalf("source=%q want /tmp", m.Source)
	}
	if m.Destination != "/data" {
		t.Fatalf("destination=%q want /data", m.Destination)
	}
	if len(m.Type.VirtioFS) == 0 {
		t.Fatal("virtiofs mount type not captured")
	}
}

func TestNetworkInfoIPv4StripsCIDR(t *testing.T) {
	n := NetworkInfo{IPv4Address: "192.168.64.7/24"}
	if got := n.IPv4(); got != "192.168.64.7" {
		t.Fatalf("IPv4()=%q want 192.168.64.7", got)
	}
	n = NetworkInfo{IPv4Address: "10.0.0.1"}
	if got := n.IPv4(); got != "10.0.0.1" {
		t.Fatalf("IPv4()=%q want 10.0.0.1", got)
	}
}

func TestParseInspectErrors(t *testing.T) {
	if _, err := parseInspect([]byte("not json")); err == nil {
		t.Fatal("want error for bad json")
	}
	if _, err := parseInspect([]byte("[]")); err == nil {
		t.Fatal("want error for empty array")
	}
	if _, err := parseInspect([]byte(`[{"configuration":{}}]`)); err == nil {
		t.Fatal("want error for missing id")
	}
}
