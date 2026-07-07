package applecontainer

import (
	"context"
	"fmt"
)

// Network represents a container network.
type Network interface {
	Remove(ctx context.Context) error
	Name() string
}

type cliNetwork struct {
	name     string
	provider *cliProvider
}

// Name returns the name of the network.
func (n *cliNetwork) Name() string {
	return n.name
}

// Remove deletes the network via the CLI.
func (n *cliNetwork) Remove(ctx context.Context) error {
	_, _, _, err := n.provider.runner.Run(ctx, []string{"network", "delete", n.name}, nil)
	return err
}

// NetworkRequest configuration.
type NetworkRequest struct {
	Name       string
	Driver     string
	Internal   bool
	EnableIPv6 bool
	Subnet     string
	SubnetV6   string
	Labels     map[string]string
}

func generateNetworkName() string {
	return randomString("apple-net-")
}

// NewNetwork creates a network.
func NewNetwork(ctx context.Context, req NetworkRequest) (Network, error) {
	if req.Name == "" {
		req.Name = generateNetworkName()
	}

	provider := newCLIProvider(Read())

	args := []string{"network", "create"}
	if req.Driver != "" {
		args = append(args, "--driver", req.Driver)
	}
	if req.Internal {
		args = append(args, "--internal")
	}
	if req.EnableIPv6 {
		args = append(args, "--ipv6")
	}
	if req.Subnet != "" {
		args = append(args, "--subnet", req.Subnet)
	}
	if req.SubnetV6 != "" {
		args = append(args, "--subnet-v6", req.SubnetV6)
	}
	for k, v := range req.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, req.Name)

	_, _, _, err := provider.runner.Run(ctx, args, nil)
	if err != nil {
		return nil, fmt.Errorf("applecontainer: failed to create network: %w", err)
	}

	return &cliNetwork{
		name:     req.Name,
		provider: provider,
	}, nil
}

// WithNetwork attaches the container to an existing network with network aliases.
func WithNetwork(aliases []string, nw Network) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		req.Networks = append(req.Networks, nw.Name())
		if req.NetworkAliases == nil {
			req.NetworkAliases = make(map[string][]string)
		}
		req.NetworkAliases[nw.Name()] = append(req.NetworkAliases[nw.Name()], aliases...)
		return nil
	}
}

// WithNewNetwork creates a network and attaches the container to it.
func WithNewNetwork(ctx context.Context, aliases []string, nwReq NetworkRequest) ContainerCustomizer {
	return func(req *ContainerRequest) error {
		nw, err := NewNetwork(ctx, nwReq)
		if err != nil {
			return err
		}
		req.Networks = append(req.Networks, nw.Name())
		if req.NetworkAliases == nil {
			req.NetworkAliases = make(map[string][]string)
		}
		req.NetworkAliases[nw.Name()] = append(req.NetworkAliases[nw.Name()], aliases...)

		req.Cleanups = append(req.Cleanups, func(ctx context.Context) error {
			return nw.Remove(ctx)
		})
		return nil
	}
}
