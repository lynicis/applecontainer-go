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
		"--rm",
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
		Networks:        []string{"net-a"},
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
	assertContains("--rm")
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
	assertContains("--network net-a")
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
