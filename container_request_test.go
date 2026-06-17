package applecontainer

import (
	"testing"
)

func TestContainerRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ContainerRequest
		wantErr bool
		msgPart string
	}{
		{
			name: "Valid request with image only",
			req: ContainerRequest{
				Image: "nginx:latest",
			},
			wantErr: false,
		},
		{
			name: "Valid request with containerfile only",
			req: ContainerRequest{
				FromContainerfile: FromContainerfile{
					Context: "./testdata",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid request: both image and containerfile set",
			req: ContainerRequest{
				Image: "nginx:latest",
				FromContainerfile: FromContainerfile{
					Context: "./testdata",
				},
			},
			wantErr: true,
			msgPart: "both Image and FromContainerfile are set",
		},
		{
			name:    "Invalid request: neither image nor containerfile set",
			req:     ContainerRequest{},
			wantErr: true,
			msgPart: "either Image or FromContainerfile must be set",
		},
		{
			name: "Valid request with multiple unique volumes and mounts",
			req: ContainerRequest{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{Source: "vol1", Target: "/data1"},
					{Source: "vol2", Target: "/data2"},
				},
				Mounts: []Mount{
					{Type: MountTypeBind, Source: "/host/path1", Target: "/data3"},
					{Type: MountTypeBind, Source: "/host/path2", Target: "/data4"},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid request: duplicate volume targets",
			req: ContainerRequest{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{Source: "vol1", Target: "/data"},
					{Source: "vol2", Target: "/data"},
				},
			},
			wantErr: true,
			msgPart: "duplicate mount target: /data",
		},
		{
			name: "Invalid request: duplicate mount targets",
			req: ContainerRequest{
				Image: "nginx:latest",
				Mounts: []Mount{
					{Type: MountTypeBind, Source: "/host/path1", Target: "/data"},
					{Type: MountTypeBind, Source: "/host/path2", Target: "/data"},
				},
			},
			wantErr: true,
			msgPart: "duplicate mount target: /data",
		},
		{
			name: "Invalid request: duplicate target between volume and mount",
			req: ContainerRequest{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{Source: "vol1", Target: "/data"},
				},
				Mounts: []Mount{
					{Type: MountTypeBind, Source: "/host/path1", Target: "/data"},
				},
			},
			wantErr: true,
			msgPart: "duplicate mount target: /data",
		},
		{
			name: "Valid request with host port mapping and exposed ports",
			req: ContainerRequest{
				Image:           "nginx:latest",
				HostPortMapping: true,
				ExposedPorts:    []string{"80/tcp"},
			},
			wantErr: false,
		},
		{
			name: "Valid request without host port mapping and empty exposed ports",
			req: ContainerRequest{
				Image:           "nginx:latest",
				HostPortMapping: false,
			},
			wantErr: false,
		},
		{
			name: "Invalid request: host port mapping enabled but exposed ports empty",
			req: ContainerRequest{
				Image:           "nginx:latest",
				HostPortMapping: true,
			},
			wantErr: true,
			msgPart: "HostPortMapping is enabled, but ExposedPorts is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.msgPart != "" {
				// verify message part
				gotMsg := err.Error()
				if !contains(gotMsg, tt.msgPart) {
					t.Errorf("Validate() error message %q does not contain expected substring %q", gotMsg, tt.msgPart)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
