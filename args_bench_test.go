package applecontainer

import "testing"

var benchArgsSink []string

func BenchmarkBuildCreateArgs(b *testing.B) {
	req := &ContainerRequest{
		Image:          "alpine:latest",
		Name:           "bench",
		Platform:       "linux/arm64",
		Env:            map[string]string{"A": "1", "B": "2", "C": "3"},
		Labels:         map[string]string{"k1": "v1", "k2": "v2"},
		ExposedPorts:   []string{"80", "443"},
		Volumes:        []VolumeMount{{Source: "/tmp/src", Target: "/data"}},
		Mounts:         []Mount{{Type: MountTypeVolume, Source: "vol", Target: "/vol"}},
		DNS:            []string{"1.1.1.1"},
		DNSDomain:      "example.com",
		DNSSearch:      []string{"example.com"},
		NoDNS:          true,
		Tmpfs:          map[string]string{"/tmp": "rw"},
		CPUs:           2,
		Memory:         512 << 20,
		ShmSize:        64 << 20,
		CapAdd:         []string{"NET_ADMIN"},
		CapDrop:        []string{"ALL"},
		Ulimits:        []Ulimit{{Name: "nofile", Soft: 1024, Hard: 2048}},
		Networks:       []string{"bench-net"},
		NetworkAliases: map[string][]string{"bench-net": {"alias1", "alias2"}},
		WorkingDir:     "/work",
		User:           "1000:1000",
		Entrypoint:     []string{"/bin/sh"},
		Cmd:            []string{"-c", "true"},
		ReadOnlyRootfs: true,
		Init:           true,
	}

	for b.Loop() {
		args, err := buildCreateArgs(req, "")
		if err != nil {
			b.Fatal(err)
		}
		benchArgsSink = args
	}
}
