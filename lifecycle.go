package applecontainer

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/lynicis/applecontainer-go/log"
)

// combineContainerHooks merges defaults and user-defined hooks.
// Pre-hooks are ordered defaults then user.
// Post-hooks are ordered user then defaults.
func combineContainerHooks(defaults, user ContainerLifecycleHooks) ContainerLifecycleHooks {
	var combined ContainerLifecycleHooks

	valDefaults := reflect.ValueOf(defaults)
	valUser := reflect.ValueOf(user)
	valCombined := reflect.ValueOf(&combined).Elem()

	t := valDefaults.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldCombined := valCombined.Field(i)
		fieldDefaults := valDefaults.Field(i)
		fieldUser := valUser.Field(i)

		if strings.HasPrefix(fieldName, "Pre") {
			// default first, then user
			combinedSlice := reflect.AppendSlice(reflect.Zero(fieldCombined.Type()), fieldDefaults)
			combinedSlice = reflect.AppendSlice(combinedSlice, fieldUser)
			fieldCombined.Set(combinedSlice)
		} else if strings.HasPrefix(fieldName, "Post") {
			// user first, then default
			combinedSlice := reflect.AppendSlice(reflect.Zero(fieldCombined.Type()), fieldUser)
			combinedSlice = reflect.AppendSlice(combinedSlice, fieldDefaults)
			fieldCombined.Set(combinedSlice)
		}
	}
	return combined
}

// defaultLoggingHooks returns lifecycle hooks that log progress to the console.
func defaultLoggingHooks(logger log.Logger) ContainerLifecycleHooks {
	if logger == nil {
		logger = log.Default()
	}
	return ContainerLifecycleHooks{
		PreBuilds: []ContainerRequestHook{
			func(ctx context.Context, req *ContainerRequest) error {
				logger.Printf("🐳 Building image from Containerfile...")
				return nil
			},
		},
		PostBuilds: []ContainerRequestHook{
			func(ctx context.Context, req *ContainerRequest) error {
				logger.Printf("🐳 Image built successfully.")
				return nil
			},
		},
		PreCreates: []ContainerRequestHook{
			func(ctx context.Context, req *ContainerRequest) error {
				logger.Printf("🐳 Creating container...")
				return nil
			},
		},
		PostCreates: []ContainerRequestHook{
			func(ctx context.Context, req *ContainerRequest) error {
				logger.Printf("🐳 Container created.")
				return nil
			},
		},
		PreStarts: []ContainerHook{
			func(ctx context.Context, c Container) error {
				logger.Printf("🐳 Starting container %s...", c.GetContainerID())
				return nil
			},
		},
		PostStarts: []ContainerHook{
			func(ctx context.Context, c Container) error {
				logger.Printf("🐳 Container %s started.", c.GetContainerID())
				return nil
			},
		},
		PostReadies: []ContainerHook{
			func(ctx context.Context, c Container) error {
				logger.Printf("🐳 Container %s is ready.", c.GetContainerID())
				return nil
			},
		},
		PreStops: []ContainerHook{
			func(ctx context.Context, c Container) error {
				logger.Printf("🐳 Stopping container %s...", c.GetContainerID())
				return nil
			},
		},
		PostStops: []ContainerHook{
			func(ctx context.Context, c Container) error {
				logger.Printf("🐳 Container %s stopped.", c.GetContainerID())
				return nil
			},
		},
		PreTerminates: []ContainerHook{
			func(ctx context.Context, c Container) error {
				logger.Printf("🐳 Terminating container %s...", c.GetContainerID())
				return nil
			},
		},
		PostTerminates: []ContainerHook{
			func(ctx context.Context, c Container) error {
				logger.Printf("🐳 Container %s terminated.", c.GetContainerID())
				return nil
			},
		},
	}
}

// defaultHooks builds the complete set of default lifecycle hooks for a container.
func defaultHooks(req *ContainerRequest, c *cliContainer) ContainerLifecycleHooks {
	var hooks ContainerLifecycleHooks

	logHooks := defaultLoggingHooks(c.log)
	hooks = combineContainerHooks(hooks, logHooks)

	hooks.PreBuilds = append(hooks.PreBuilds, func(ctx context.Context, r *ContainerRequest) error {
		if r.FromContainerfile.Context != "" {
			return defaultBuildHook(ctx, r, c)
		}
		return nil
	})

	hooks.PreCreates = append(hooks.PreCreates, func(ctx context.Context, r *ContainerRequest) error {
		if c.id == "" {
			created, err := c.provider.CreateContainer(ctx, r)
			if err != nil {
				return err
			}
			c.id = created.id
		}
		return nil
	})

	hooks.PostCreates = append(hooks.PostCreates, func(ctx context.Context, r *ContainerRequest) error {
		for _, file := range r.Files {
			content, err := os.ReadFile(file.HostFilePath)
			if err != nil {
				return fmt.Errorf("applecontainer: failed to read host file %s for copy: %w", file.HostFilePath, err)
			}
			if err := c.CopyToContainer(ctx, content, file.ContainerFilePath, file.FileMode); err != nil {
				return fmt.Errorf("applecontainer: failed to copy file %s to container: %w", file.HostFilePath, err)
			}
		}
		return nil
	})

	hooks.PostStarts = append(hooks.PostStarts, func(ctx context.Context, ctr Container) error {
		if c.logFanout == nil {
			c.logFanout = &logFanout{}
		}
		return c.logFanout.Start(ctx, c)
	})

	hooks.PostStarts = append(hooks.PostStarts, func(ctx context.Context, ctr Container) error {
		if r := c.req; r.WaitingFor != nil {
			if err := r.WaitingFor.WaitUntilReady(ctx, waitTarget{c}); err != nil {
				return fmt.Errorf("applecontainer: wait strategy failed: %w", err)
			}
		}
		c.isRunning.Store(true)
		return nil
	})

	hooks.PostStops = append(hooks.PostStops, func(ctx context.Context, ctr Container) error {
		if c.logFanout != nil {
			_ = c.logFanout.Stop()
		}
		c.isRunning.Store(false)
		return nil
	})

	return hooks
}

// executeLifecycle runs the container lifecycle phases.
func (c *cliContainer) executeLifecycle(ctx context.Context, isStart bool) error {
	if isStart {
		for _, l := range c.lifecycle {
			for _, hook := range l.PreBuilds {
				if err := hook(ctx, &c.req); err != nil {
					return err
				}
			}
		}
		for _, l := range c.lifecycle {
			for _, hook := range l.PostBuilds {
				if err := hook(ctx, &c.req); err != nil {
					return err
				}
			}
		}
		for _, l := range c.lifecycle {
			for _, hook := range l.PreCreates {
				if err := hook(ctx, &c.req); err != nil {
					return err
				}
			}
		}
		for _, l := range c.lifecycle {
			for _, hook := range l.PostCreates {
				if err := hook(ctx, &c.req); err != nil {
					return err
				}
			}
		}
		for _, l := range c.lifecycle {
			for _, hook := range l.PreStarts {
				if err := hook(ctx, c); err != nil {
					return err
				}
			}
		}
		if err := c.provider.StartContainer(ctx, c); err != nil {
			return err
		}
		for _, l := range c.lifecycle {
			for _, hook := range l.PostStarts {
				if err := hook(ctx, c); err != nil {
					return err
				}
			}
		}
		for _, l := range c.lifecycle {
			for _, hook := range l.PostReadies {
				if err := hook(ctx, c); err != nil {
					return err
				}
			}
		}
	} else {
		for _, l := range c.lifecycle {
			for _, hook := range l.PreStops {
				if err := hook(ctx, c); err != nil {
					return err
				}
			}
		}
		for _, l := range c.lifecycle {
			for _, hook := range l.PostStops {
				if err := hook(ctx, c); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type waitTarget struct {
	*cliContainer
}

func (w waitTarget) Exec(ctx context.Context, cmd []string, opts ...any) (int, []byte, error) {
	return w.cliContainer.Exec(ctx, cmd)
}

func (w waitTarget) StateStatus(ctx context.Context) (string, error) {
	return w.cliContainer.StateStatus(ctx)
}

func (w waitTarget) StateExitCode(ctx context.Context) (int, error) {
	return w.cliContainer.StateExitCode(ctx)
}

func (w waitTarget) Logs(ctx context.Context) (io.ReadCloser, error) {
	if w.logFanout != nil && w.logFanout.started {
		return w.logFanout.NewReader(ctx)
	}
	return w.provider.ContainerLogs(ctx, w.id, true, 0)
}

func (w waitTarget) CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error) {
	return w.cliContainer.CopyFileFromContainer(ctx, path)
}
