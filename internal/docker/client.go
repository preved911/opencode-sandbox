// Package docker builds a Docker SDK client and derives the host name used
// in the printed "opencode attach http://<host>:<port>" line.
package docker

import (
	"net/url"
	"os"

	"github.com/docker/docker/client"
)

// NewClient builds a Docker client.
//
// If host is non-empty, it overrides DOCKER_HOST. Otherwise the standard
// environment variables (DOCKER_HOST, DOCKER_API_VERSION, DOCKER_CERT_PATH,
// DOCKER_TLS_VERIFY) are consulted.
func NewClient(host string) (*client.Client, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if host != "" {
		opts = append(opts, client.WithHost(host))
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
	return os.Getenv("DOCKER_HOST")
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
