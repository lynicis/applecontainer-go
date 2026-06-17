package applecontainer

// WithExecUser sets the user for exec command execution.
func WithExecUser(user string) ProcessOption {
	return func(o *processOptions) {
		o.User = user
	}
}

// WithExecWorkingDir sets the working directory for exec command execution.
func WithExecWorkingDir(dir string) ProcessOption {
	return func(o *processOptions) {
		o.WorkingDir = dir
	}
}

// WithExecEnv sets the environment variables for exec command execution.
func WithExecEnv(env []string) ProcessOption {
	return func(o *processOptions) {
		o.Env = env
	}
}
