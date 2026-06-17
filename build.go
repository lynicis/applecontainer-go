package applecontainer

import (
	"context"
	"fmt"
)

func generateBuildTag() string {
	return randomString("applecontainer-")
}

func defaultBuildHook(ctx context.Context, r *ContainerRequest, c *cliContainer) error {
	cf := r.FromContainerfile
	if cf.Context == "" {
		return nil
	}

	tag := ""
	if len(cf.Tags) > 0 && cf.Tags[0] != "" {
		tag = cf.Tags[0]
	} else {
		tag = generateBuildTag()
	}

	args := []string{"build", "-t", tag, "--progress", "plain"}
	if cf.File != "" {
		args = append(args, "-f", cf.File)
	}
	for k, v := range cf.BuildArgs {
		if v != nil {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, *v))
		}
	}
	if cf.Target != "" {
		args = append(args, "--target", cf.Target)
	}
	if cf.NoCache {
		args = append(args, "--no-cache")
	}
	if cf.Pull {
		args = append(args, "--pull")
	}
	if cf.Platform != "" {
		args = append(args, "--platform", cf.Platform)
	}
	for k, v := range cf.Secrets {
		args = append(args, "--secret", fmt.Sprintf("id=%s,src=%s", k, v))
	}
	args = append(args, cf.Context)

	_, _, _, err := c.provider.runner.Run(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("applecontainer: build failed: %w", err)
	}

	r.Image = tag
	c.image = tag

	if !cf.KeepImage {
		for i := range c.lifecycle {
			c.lifecycle[i].PostTerminates = append(c.lifecycle[i].PostTerminates, func(ctx context.Context, ctr Container) error {
				_, _, _, err := c.provider.runner.Run(ctx, []string{"image", "delete", tag}, nil)
				return err
			})
		}
	}

	return nil
}
