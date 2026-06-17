package applecontainer

import (
	"context"
	"fmt"
)

// Volume represents a container volume.
type Volume interface {
	Remove(ctx context.Context) error
	Name() string
}

type cliVolume struct {
	name     string
	provider *cliProvider
}

// Name returns the name of the volume.
func (v *cliVolume) Name() string {
	return v.name
}

// Remove deletes the volume via the CLI.
func (v *cliVolume) Remove(ctx context.Context) error {
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

// VolumeOption sets volume request parameters.
type VolumeOption func(*VolumeRequest)

// WithVolumeNameOption sets the volume name.
func WithVolumeNameOption(name string) VolumeOption {
	return func(r *VolumeRequest) {
		r.Name = name
	}
}

// WithVolumeLabels sets labels on the volume.
func WithVolumeLabels(labels map[string]string) VolumeOption {
	return func(r *VolumeRequest) {
		if r.Labels == nil {
			r.Labels = make(map[string]string)
		}
		for k, v := range labels {
			r.Labels[k] = v
		}
	}
}

// WithVolumeSize sets the size of the volume.
func WithVolumeSize(size string) VolumeOption {
	return func(r *VolumeRequest) {
		r.Size = size
	}
}

// WithVolumeOpt adds options to the volume creation.
func WithVolumeOpt(key, val string) VolumeOption {
	return func(r *VolumeRequest) {
		if r.Options == nil {
			r.Options = make(map[string]string)
		}
		r.Options[key] = val
	}
}

func generateVolumeName() string {
	return randomString("apple-vol-")
}

// NewVolume creates a volume.
func NewVolume(ctx context.Context, opts ...VolumeOption) (Volume, error) {
	req := VolumeRequest{}
	for _, o := range opts {
		o(&req)
	}
	if req.Name == "" {
		req.Name = generateVolumeName()
	}

	provider := newCLIProvider(Read())

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

	return &cliVolume{
		name:     req.Name,
		provider: provider,
	}, nil
}
