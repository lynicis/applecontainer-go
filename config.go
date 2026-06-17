package applecontainer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds resolved configuration for the applecontainer-go library.
// Read once (sync.Once) from ~/.applecontainer.properties + env on first use.
type Config struct {
	BinaryPath      string
	Debug           bool
	DefaultNetwork  string
	DefaultPlatform string
	HubImagePrefix  string
	PullTimeout     time.Duration
}

var (
	configOnce     sync.Once
	configVal      Config
	propertiesPath = defaultPropertiesPath()
)

func defaultPropertiesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".applecontainer.properties")
}

// Read returns the singleton Config, reading ~/.applecontainer.properties and
// environment variables once. Env values override properties-file values.
func Read() Config {
	configOnce.Do(func() {
		configVal = readConfig()
	})
	return configVal
}

// Reset clears the cached Config. For tests only.
func Reset() {
	configOnce = sync.Once{}
	configVal = Config{}
}

func readConfig() Config {
	c := Config{
		DefaultNetwork: "default",
		PullTimeout:    5 * time.Minute,
	}
	if propertiesPath != "" {
		if b, err := os.ReadFile(propertiesPath); err == nil {
			applyProperties(&c, string(b))
		}
	}
	applyEnv(&c)
	if c.BinaryPath == "" {
		if p, err := exec.LookPath("container"); err == nil {
			c.BinaryPath = p
		} else {
			c.BinaryPath = "container"
		}
	}
	return c
}

func applyProperties(c *Config, content string) {
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "container.binary.path":
			c.BinaryPath = v
		case "container.default.network":
			c.DefaultNetwork = v
		case "container.default.platform":
			c.DefaultPlatform = v
		case "hub.image.name.prefix":
			c.HubImagePrefix = v
		case "container.pull.timeout":
			if d, err := time.ParseDuration(v); err == nil {
				c.PullTimeout = d
			}
		case "container.debug":
			c.Debug = parseBool(v)
		}
	}
}

func applyEnv(c *Config) {
	if v, ok := os.LookupEnv("CONTAINER_BINARY"); ok {
		c.BinaryPath = v
	}
	if v, ok := os.LookupEnv("CONTAINER_DEBUG"); ok {
		c.Debug = parseBool(v)
	}
	if v, ok := os.LookupEnv("CONTAINER_DEFAULT_PLATFORM"); ok {
		c.DefaultPlatform = v
	}
	if v, ok := os.LookupEnv("APPLECONTAINER_HUB_IMAGE_NAME_PREFIX"); ok {
		c.HubImagePrefix = v
	}
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

// runner returns the commandRunner used to invoke the container binary.
func (c Config) runner() commandRunner { return newExecRunner(c.BinaryPath) }

// VersionCheck runs `container --version`, parses the output, and verifies the
// version is >= 1.0.0. Called lazily on first Run().
func VersionCheck(ctx context.Context) (string, error) {
	r := Read().runner()
	if providerRunnerOverride != nil {
		r = providerRunnerOverride
	}
	return checkVersion(ctx, r)
}

func checkVersion(ctx context.Context, r commandRunner) (string, error) {
	out, _, _, err := r.Run(ctx, []string{"--version"}, nil)
	if err != nil {
		return "", fmt.Errorf("applecontainer: cannot run container CLI (is it installed? macOS 26, Apple silicon required): %w", err)
	}
	ver, err := parseVersion(string(out))
	if err != nil {
		return "", fmt.Errorf("applecontainer: cannot parse %q: %w", strings.TrimSpace(string(out)), err)
	}
	if !versionAtLeast(ver, "1.0.0") {
		return ver, fmt.Errorf("applecontainer: container CLI version %s is older than required 1.0.0; please upgrade", ver)
	}
	return ver, nil
}

// parseVersion extracts the semver from "container version 1.0.0 (build: ...)".
func parseVersion(s string) (string, error) {
	fields := strings.Fields(s)
	for i, f := range fields {
		if f == "version" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", errors.New("no 'version' token in output")
}

// versionAtLeast reports whether got >= min as semver (major.minor.patch).
func versionAtLeast(got, min string) bool {
	g := parseSemver(got)
	m := parseSemver(min)
	if g[0] != m[0] {
		return g[0] > m[0]
	}
	if g[1] != m[1] {
		return g[1] > m[1]
	}
	return g[2] >= m[2]
}

func parseSemver(s string) [3]int {
	var v [3]int
	parts := strings.SplitN(s, ".", 3)
	for i := 0; i < len(parts) && i < 3; i++ {
		digits := ""
		for _, r := range parts[i] {
			if r >= '0' && r <= '9' {
				digits += string(r)
			} else {
				break
			}
		}
		v[i], _ = strconv.Atoi(digits)
	}
	return v
}
