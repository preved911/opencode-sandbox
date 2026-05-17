// Package config defines the sandbox configuration schema and loader.
//
// A config file may be either:
//   - a single sandbox, with top-level name/docker/build/run keys, or
//   - a profiles file, with a top-level "profiles" map keyed by name.
//
// Load resolves an explicit path, then ./container-sandbox.yaml, then the
// central default at $HOME/.config/opencode-sandbox/config.yaml
// ($XDG_CONFIG_HOME/opencode-sandbox/config.yaml when XDG_CONFIG_HOME is set).
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/preved911/container-sandbox/internal/paths"
)

// Config is a single sandbox configuration.
type Config struct {
	Name   string       `yaml:"name,omitempty"`
	Docker DockerConfig `yaml:"docker,omitempty"`
	Build  BuildConfig  `yaml:"build,omitempty"`
	Run    RunConfig    `yaml:"run,omitempty"`

	baseDir string
}

type DockerConfig struct {
	Host       string `yaml:"host,omitempty"`
	AttachHost string `yaml:"attach_host,omitempty"`
}

type BuildConfig struct {
	Image      string            `yaml:"image,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Context    string            `yaml:"context,omitempty"`
	Target     string            `yaml:"target,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
	Secrets    []Secret          `yaml:"secrets,omitempty"`
	Pull       bool              `yaml:"pull,omitempty"`
}

// Secret is a BuildKit-style build secret. Exactly one of Src or Env must be set.
type Secret struct {
	ID  string `yaml:"id"`
	Src string `yaml:"src,omitempty"`
	Env string `yaml:"env,omitempty"`
}

type RunConfig struct {
	Env     map[string]string `yaml:"env,omitempty"`
	Mounts  []Mount           `yaml:"mounts,omitempty"`
	Workdir string            `yaml:"workdir,omitempty"`
	User    string            `yaml:"user,omitempty"`
	Port    PortConfig        `yaml:"port,omitempty"`
}

type Mount struct {
	Source   string `yaml:"source,omitempty"`
	Target   string `yaml:"target"`
	Type     string `yaml:"type,omitempty"`
	ReadOnly bool   `yaml:"readonly,omitempty"`
}

type PortConfig struct {
	Bind string `yaml:"bind,omitempty"`
}

// file is the on-disk shape; either profiles or the inline single-config fields are used.
type file struct {
	Profiles map[string]*Config `yaml:"profiles,omitempty"`

	Name   string       `yaml:"name,omitempty"`
	Docker DockerConfig `yaml:"docker,omitempty"`
	Build  BuildConfig  `yaml:"build,omitempty"`
	Run    RunConfig    `yaml:"run,omitempty"`
}

// Load resolves the right config file and returns the selected sandbox.
//
// Resolution order:
//  1. explicitPath (-c flag) — profile applied if the file contains profiles
//  2. ./container-sandbox.yaml — project-local override
//  3. $HOME/.config/opencode-sandbox/config.yaml — central default
func Load(explicitPath, profile string) (*Config, error) {
	if explicitPath != "" {
		return loadFile(explicitPath, profile)
	}
	const localName = "container-sandbox.yaml"
	if _, err := os.Stat(localName); err == nil {
		return loadFile(localName, profile)
	}
	central, err := centralConfigPath()
	if err != nil {
		return nil, err
	}
	return loadFile(central, profile)
}

func centralConfigPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "opencode-sandbox", "config.yaml"), nil
}

func loadFile(path, profile string) (*Config, error) {
	abs, err := paths.Expand(path, "")
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", abs, err)
	}
	var f file
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", abs, err)
	}
	baseDir := filepath.Dir(abs)

	if len(f.Profiles) > 0 {
		if profile == "" {
			return nil, fmt.Errorf("config %s defines profiles; select one with -p/--profile", abs)
		}
		c, ok := f.Profiles[profile]
		if !ok {
			return nil, fmt.Errorf("profile %q not found in %s", profile, abs)
		}
		if c.Name == "" {
			c.Name = profile
		}
		c.baseDir = baseDir
		return c, nil
	}

	c := &Config{
		Name:    f.Name,
		Docker:  f.Docker,
		Build:   f.Build,
		Run:     f.Run,
		baseDir: baseDir,
	}
	if c.Name == "" {
		if cwd, err := os.Getwd(); err == nil {
			c.Name = filepath.Base(cwd)
		}
	}
	return c, nil
}

// BaseDir is the directory relative paths in this config are resolved against.
func (c *Config) BaseDir() string { return c.baseDir }
