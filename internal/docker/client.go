// Package docker builds a Docker SDK client and derives the host name used
// in the printed "opencode attach http://<host>:<port>" line.
package docker

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
)

// NewClient builds a Docker client.
//
// Host resolution order:
//  1. host argument (from config docker.host or --docker-host flag)
//  2. DOCKER_HOST environment variable
//  3. active Docker CLI context (DOCKER_CONTEXT env or currentContext in ~/.docker/config.json)
//  4. SDK default (platform socket)
func NewClient(host string) (*client.Client, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	} else if h := resolvedHost(); h != "" {
		opts = append(opts, client.WithHost(h))
	} else {
		opts = append(opts, client.FromEnv)
	}
	return client.NewClientWithOpts(opts...)
}

// EffectiveHost reports the Docker host string that the client will actually
// use, applying the same precedence as NewClient.
func EffectiveHost(configHost string) string {
	if configHost != "" {
		return configHost
	}
	return resolvedHost()
}

// resolvedHost returns the Docker host from the environment or the active
// Docker CLI context, whichever is set first.
func resolvedHost() string {
	if h := os.Getenv("DOCKER_HOST"); h != "" {
		return h
	}
	return activeContextHost()
}

// activeContextHost reads the active Docker CLI context and returns its
// daemon endpoint. It honours the DOCKER_CONTEXT env var the same way the
// Docker CLI does.
func activeContextHost() string {
	configDir := dockerConfigDir()

	name := os.Getenv("DOCKER_CONTEXT")
	if name == "" {
		name = currentContextName(configDir)
	}
	if name == "" || name == "default" {
		return ""
	}
	return contextEndpointHost(configDir, name)
}

// dockerConfigDir returns the Docker CLI configuration directory.
func dockerConfigDir() string {
	if d := os.Getenv("DOCKER_CONFIG"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docker")
}

// currentContextName reads currentContext from ~/.docker/config.json.
func currentContextName(configDir string) string {
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		return ""
	}
	var cfg struct {
		CurrentContext string `json:"currentContext"`
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg.CurrentContext
}

// contextEndpointHost returns the docker endpoint Host for a named context.
// Docker stores context metadata at
// ~/.docker/contexts/meta/<sha256(name)>/meta.json.
func contextEndpointHost(configDir, name string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(name)))
	data, err := os.ReadFile(filepath.Join(configDir, "contexts", "meta", hash, "meta.json"))
	if err != nil {
		return ""
	}
	var meta struct {
		Endpoints struct {
			Docker struct {
				Host string `json:"Host"`
			} `json:"docker"`
		} `json:"Endpoints"`
	}
	_ = json.Unmarshal(data, &meta)
	return meta.Endpoints.Docker.Host
}

// IsRemoteHost reports whether host refers to a remote Docker daemon
// (TCP, SSH, HTTP/HTTPS) rather than a local socket.
func IsRemoteHost(host string) bool {
	if host == "" {
		return false
	}
	u, err := url.Parse(host)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "tcp", "ssh", "http", "https":
		return true
	default:
		return false
	}
}

// AttachHost derives the host portion of the attach URL from a Docker host URL.
//
//	tcp://1.2.3.4:2375       → 1.2.3.4
//	ssh://user@box.internal  → box.internal
//	unix:// / npipe:// / "" → 127.0.0.1
//
// A wildcard (0.0.0.0, ::) collapses to 127.0.0.1 so the printed URL is
// actually dialable.
func AttachHost(dockerHost string) string {
	const local = "127.0.0.1"
	if dockerHost == "" {
		return local
	}
	u, err := url.Parse(dockerHost)
	if err != nil {
		return local
	}
	switch u.Scheme {
	case "tcp", "ssh", "http", "https":
		h := u.Hostname()
		if h == "" || h == "0.0.0.0" || h == "::" {
			return local
		}
		return h
	default:
		return local
	}
}
