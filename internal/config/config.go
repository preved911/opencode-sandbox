// Package config defines the sandbox configuration schema and loader.
//
// Config files always use a profiles format:
//
//	docker:
//	  host: tcp://...         # global docker host; overrides DOCKER_HOST for all profiles
//	default_profile: go-dev   # used when -p/--profile is omitted
//
//	profiles:
//	  go-dev:
//	    build: ...
//	    run: ...
//
// Load resolves an explicit path, then ./opencode-sandbox.yaml, then the
// central default at $HOME/.config/opencode-sandbox/config.yaml
// ($XDG_CONFIG_HOME/opencode-sandbox/config.yaml when XDG_CONFIG_HOME is set).
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/preved911/opencode-sandbox/internal/paths"
)

// Config is a single sandbox/profile configuration.
type Config struct {
	Name       string       `yaml:"name,omitempty"`
	DockerHost string       // effective docker host; set by loader, overridable by CLI flag
	Docker     DockerConfig `yaml:"docker,omitempty"`
	Build      BuildConfig  `yaml:"build,omitempty"`
	Run        RunConfig    `yaml:"run,omitempty"`

	baseDir string
}

// DockerConfig holds per-profile docker settings.
type DockerConfig struct {
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

// globalDockerConfig holds docker settings at the file/global scope.
type globalDockerConfig struct {
	Host string `yaml:"host,omitempty"`
}

// file is the on-disk shape.
type file struct {
	Docker         globalDockerConfig `yaml:"docker,omitempty"`
	DefaultProfile string             `yaml:"default_profile,omitempty"`
	Profiles       map[string]*Config `yaml:"profiles"`
}

// Load resolves the right config file and returns the selected sandbox.
//
// Resolution order for the config file:
//  1. explicitPath (-c flag)
//  2. ./opencode-sandbox.yaml
//  3. $HOME/.config/opencode-sandbox/config.yaml
//
// Profile selection within that file:
//  1. profile (-p/--profile flag)
//  2. default_profile in the file
//  3. the sole profile if only one is defined
func Load(explicitPath, profile string) (*Config, error) {
	if explicitPath != "" {
		return loadFile(explicitPath, profile)
	}
	const localName = "opencode-sandbox.yaml"
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

	if len(f.Profiles) == 0 {
		return nil, fmt.Errorf("config %s: no profiles defined", abs)
	}

	name := profile
	if name == "" {
		name = f.DefaultProfile
	}
	if name == "" {
		if len(f.Profiles) == 1 {
			for k := range f.Profiles {
				name = k
			}
		} else {
			return nil, fmt.Errorf("config %s defines %d profiles; select one with -p/--profile or set default_profile", abs, len(f.Profiles))
		}
	}

	c, ok := f.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in %s", name, abs)
	}
	if c.Name == "" {
		c.Name = name
	}
	c.DockerHost = f.Docker.Host
	c.baseDir = baseDir
	return c, nil
}

// BaseDir is the directory relative paths in this config are resolved against.
func (c *Config) BaseDir() string { return c.baseDir }
