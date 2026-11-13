package applecontainer

import (
	"strconv"
	"strings"
	"testing"
)

func TestBuildCreateArgs_Minimal(t *testing.T) {
	req := &ContainerRequest{
		Image: "nginx:latest",
	}

	args, err := buildCreateArgs(req, "/tmp/cidfile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrefix := []string{
		"create",
		"--cidfile", "/tmp/cidfile",
		"-l", "applecontainer=true",
		"-l", "applecontainer.session=" + SessionID(),
		"nginx:latest",
	}

	if len(args) != len(expectedPrefix) {
		t.Fatalf("expected args length %d, got %d: %v", len(expectedPrefix), len(args), args)
	}

	for i, v := range expectedPrefix {
		if args[i] != v {
			t.Errorf("expected arg at index %d to be %q, got %q", i, v, args[i])
		}
	}
}

func TestBuildCreateArgs_Full(t *testing.T) {
	req := &ContainerRequest{
		Image:           "ubuntu:latest",
		Name:            "test-container",
		Platform:        "linux/arm64",
		Arch:            "arm64",
		OS:              "linux",
		Rosetta:         true,
		Init:            true,
		WorkingDir:      "/workspace",
		User:            "user1",
		Entrypoint:      []string{"/bin/entrypoint"},
		Cmd:             []string{"run", "arg1"},
		CPUs:            1.5,
		Memory:          512 * 1024 * 1024,
		ReadOnlyRootfs:  true,
		ShmSize:         64 * 1024 * 1024,
		Env:             map[string]string{"ENV1": "VAL1"},
		Labels:          map[string]string{"LABEL1": "VAL2"},
		CapAdd:          []string{"SYS_ADMIN"},
		CapDrop:         []string{"NET_ADMIN"},
		DNS:             []string{"1.1.1.1", "8.8.8.8"},
		DNSDomain:       "local",
		DNSSearch:       []string{"example.com"},
		NoDNS:           true,
		Ulimits:         []Ulimit{{Name: "nofile", Soft: 1, Hard: 2}},
		Networks:        []string{"net-a"},
		NetworkAliases:  map[string][]string{"net-a": {"web"}},
		Volumes:         []VolumeMount{{Source: "v1", Target: "/v1", ReadOnly: true}},
		Mounts:          []Mount{{Type: MountTypeBind, Source: "/src", Target: "/dst"}},
		Tmpfs:           map[string]string{"/tmp": "size=32m"},
		HostPortMapping: true,
		ExposedPorts:    []string{"80/tcp"},
	}

	args, err := buildCreateArgs(req, "/tmp/cidfile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argStr := strings.Join(args, " ")

	// Helper to assert substring presence
	assertContains := func(sub string) {
		if !strings.Contains(argStr, sub) {
			t.Errorf("expected args to contain %q, got %q", sub, argStr)
		}
	}

	assertContains("create")
	assertContains("--cidfile /tmp/cidfile")
	assertContains("applecontainer.session=" + SessionID())
	assertContains("--name test-container")
	assertContains("--platform linux/arm64")
	assertContains("--arch arm64")
	assertContains("--os linux")
	assertContains("--rosetta")
	assertContains("--init")
	assertContains("-w /workspace")
	assertContains("-u user1")
	assertContains("--entrypoint /bin/entrypoint")
	assertContains("-c 1.5")
	assertContains("-m 536870912")
	assertContains("--read-only")
	assertContains("--shm-size 67108864")
	assertContains("-e ENV1=VAL1")
	assertContains("-l LABEL1=VAL2")
	assertContains("--cap-add SYS_ADMIN")
	assertContains("--cap-drop NET_ADMIN")
	assertContains("--dns 1.1.1.1")
	assertContains("--dns 8.8.8.8")
	assertContains("--dns-domain local")
	assertContains("--dns-search example.com")
	assertContains("--no-dns")
	assertContains("--ulimit nofile=1:2")
	assertContains("--network net-a,alias=web")
	assertContains("-v v1:/v1:ro")
	assertContains("--mount type=bind,source=/src,target=/dst")
	assertContains("--tmpfs /tmp:size=32m")
	assertContains("ubuntu:latest run arg1")

	// Host port mapping checks
	hostPort := req.HostPorts["80/tcp"]
	if hostPort <= 0 {
		t.Errorf("expected host port for 80/tcp to be allocated, got %d", hostPort)
	}
	assertContains("--publish " + strconv.Itoa(hostPort) + ":80/tcp")
}

func TestBuildCreateArgs_EntrypointArgsAndHostPortKeys(t *testing.T) {
	for _, key := range []string{"8080", "8080/tcp"} {
		t.Run(key, func(t *testing.T) {
			req := &ContainerRequest{
				Image:           "img",
				Entrypoint:      []string{"/bin/sh", "-c"},
				Cmd:             []string{"echo", "hi"},
				HostPortMapping: true,
				ExposedPorts:    []string{"8080/tcp"},
				HostPorts:       map[string]int{key: 18080},
			}

			args, err := buildCreateArgs(req, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			imgIdx := -1
			for i, a := range args {
				if a == "img" {
					imgIdx = i
					break
				}
			}
			if imgIdx < 0 {
				t.Fatalf("image argument missing: %v", args)
			}
			wantTail := []string{"-c", "echo", "hi"}
			if got := args[imgIdx+1:]; strings.Join(got, " ") != strings.Join(wantTail, " ") {
				t.Errorf("got tail %v, want %v", got, wantTail)
			}
			if !strings.Contains(strings.Join(args, " "), "--publish 18080:8080/tcp") {
				t.Errorf("expected fixed host port publish, got %v", args)
			}
		})
	}
}

func TestBuildCreateArgs_UlimitForms(t *testing.T) {
	req := &ContainerRequest{
		Image: "img",
		Ulimits: []Ulimit{
			{Name: "pair", Soft: 1, Hard: 2},
			{Name: "hardonly", Soft: 0, Hard: 3},
			{Name: "softonly", Soft: 4},
		},
	}

	args, err := buildCreateArgs(req, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argStr := strings.Join(args, " ")
	for _, want := range []string{"--ulimit pair=1:2", "--ulimit hardonly=3", "--ulimit softonly=4"} {
		if !strings.Contains(argStr, want) {
			t.Errorf("expected %q in args, got %v", want, args)
		}
	}
}

func TestBuildPullArgs(t *testing.T) {
	req := &ContainerRequest{
		Image:    "nginx:latest",
		Platform: "linux/arm64",
		Arch:     "arm64",
		OS:       "linux",
	}

	want := []string{"image", "pull", "--platform", "linux/arm64", "--arch", "arm64", "--os", "linux", "nginx:latest"}
	got := buildPullArgs(req)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := buildPullArgs(&ContainerRequest{Image: "alpine"}); strings.Join(got, " ") != "image pull alpine" {
		t.Errorf("got %v, want image pull alpine", got)
	}
}
