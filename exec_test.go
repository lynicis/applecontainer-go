package applecontainer

import "testing"

func TestProcessOptions(t *testing.T) {
	var opts processOptions
	WithExecUser("user1")(&opts)
	WithExecWorkingDir("/var/tmp")(&opts)
	WithExecEnv([]string{"A=B"})(&opts)

	if opts.User != "user1" {
		t.Errorf("expected User 'user1', got %q", opts.User)
	}
	if opts.WorkingDir != "/var/tmp" {
		t.Errorf("expected WorkingDir '/var/tmp', got %q", opts.WorkingDir)
	}
	if len(opts.Env) != 1 || opts.Env[0] != "A=B" {
		t.Errorf("expected Env ['A=B'], got %v", opts.Env)
	}
}
