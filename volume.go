package applecontainer

import (
	"context"
	"fmt"
)

type Volume struct {
	name     string
	provider *Provider
}

// Name returns the name of the volume.
func (v *Volume) Name() string {
	return v.name
}

// Remove deletes the volume via the CLI.
func (v *Volume) Remove(ctx context.Context) error {
	_, _, _, err := v.provider.runner.Run(ctx, []string{"volume", "delete", v.name}, nil)
	return err
}

// VolumeRequest configuration.
type VolumeRequest struct {
	Name    string
	Size    string
	Labels  map[string]string
	Options map[string]string
}

func generateVolumeName() string {
	return randomString("apple-vol-")
}

// NewVolume creates a volume.
func NewVolume(ctx context.Context, req VolumeRequest) (*Volume, error) {
	if req.Name == "" {
		req.Name = generateVolumeName()
	}

	provider := newProvider(Read())

	args := []string{"volume", "create"}
	if req.Size != "" {
		args = append(args, "--size", req.Size)
	}
	for k, v := range req.Options {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range req.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, req.Name)

	_, _, _, err := provider.runner.Run(ctx, args, nil)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to create volume: %w", err)
	}

	return &Volume{
		name:     req.Name,
		provider: provider,
	}, nil
}
