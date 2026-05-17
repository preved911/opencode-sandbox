// Package run creates and starts a sandbox container, then reports the host
// port that Docker assigned to the published container port.
package run

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/preved911/container-sandbox/internal/config"
	"github.com/preved911/container-sandbox/internal/paths"
	"github.com/preved911/container-sandbox/internal/sandbox"
)

// containerPort is the port opencode serves on inside the container.
const containerPort nat.Port = "4096/tcp"

var entrypoint = []string{"opencode", "serve", "--host=0.0.0.0", "--port=4096"}

// Result describes a successfully started sandbox.
type Result struct {
	ContainerID string
	Name        string
	HostPort    int
}

// Start creates and starts a container named name running image.
func Start(ctx context.Context, cli *client.Client, cfg *config.Config, image, name string) (*Result, error) {
	envSlice := make([]string, 0, len(cfg.Run.Env))
	for k, v := range cfg.Run.Env {
		envSlice = append(envSlice, k+"="+v)
	}

	mounts, err := buildMounts(cfg)
	if err != nil {
		return nil, err
	}

	bindIP := cfg.Run.Port.Bind
	if bindIP == "" {
		bindIP = "127.0.0.1"
	}

	cConf := &container.Config{
		Image:        image,
		Entrypoint:   entrypoint,
		Env:          envSlice,
		WorkingDir:   cfg.Run.Workdir,
		User:         cfg.Run.User,
		ExposedPorts: nat.PortSet{containerPort: struct{}{}},
		Labels: map[string]string{
			sandbox.Label:     "true",
			sandbox.LabelName: cfg.Name,
		},
	}
	hConf := &container.HostConfig{
		Mounts: mounts,
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{{HostIP: bindIP, HostPort: "0"}},
		},
	}

	created, err := cli.ContainerCreate(ctx, cConf, hConf, nil, nil, name)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}
	if err := cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		_ = cli.ContainerRemove(ctx, created.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("start container: %w", err)
	}

	inspect, err := cli.ContainerInspect(ctx, created.ID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}
	if inspect.NetworkSettings == nil {
		return nil, fmt.Errorf("container has no network settings yet")
	}
	bindings := inspect.NetworkSettings.Ports[containerPort]
	if len(bindings) == 0 || bindings[0].HostPort == "" {
		return nil, fmt.Errorf("container did not publish %s", containerPort)
	}
	port, err := strconv.Atoi(bindings[0].HostPort)
	if err != nil {
		return nil, fmt.Errorf("parse host port %q: %w", bindings[0].HostPort, err)
	}

	return &Result{
		ContainerID: created.ID,
		Name:        name,
		HostPort:    port,
	}, nil
}

func buildMounts(cfg *config.Config) ([]mount.Mount, error) {
	out := make([]mount.Mount, 0, len(cfg.Run.Mounts))
	for i, m := range cfg.Run.Mounts {
		if m.Target == "" {
			return nil, fmt.Errorf("mount %d: target is required", i)
		}
		var mt mount.Type
		switch m.Type {
		case "", "bind":
			mt = mount.TypeBind
		case "volume":
			mt = mount.TypeVolume
		case "tmpfs":
			mt = mount.TypeTmpfs
		default:
			return nil, fmt.Errorf("mount %d: unknown type %q", i, m.Type)
		}
		mm := mount.Mount{
			Type:     mt,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		}
		switch mt {
		case mount.TypeBind:
			if m.Source == "" {
				return nil, fmt.Errorf("mount %d: bind source is required", i)
			}
			abs, err := paths.Expand(m.Source, cfg.BaseDir())
			if err != nil {
				return nil, fmt.Errorf("mount %d source: %w", i, err)
			}
			mm.Source = abs
		case mount.TypeVolume:
			// Source may be empty (anonymous volume) or a named volume.
			mm.Source = m.Source
		}
		out = append(out, mm)
	}
	return out, nil
}
