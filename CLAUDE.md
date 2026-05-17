# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build the binary
go build ./cmd/opencode-sandbox/

# Build and verify all packages compile
go build ./...

# Run a specific test
go test ./internal/config/...

# Run all tests
go test ./...
```

There is no Makefile. The binary entry point is `cmd/opencode-sandbox/main.go`.

## Architecture

The tool builds and runs Docker containers that serve an opencode `serve` endpoint, then prints `opencode attach http://<host>:<port>` so a local client can connect.

### Config loading (`internal/config`)

Config files are always **profiles-based**:

```yaml
docker:
  host: tcp://...          # global; forwarded as DOCKER_HOST to all subprocesses
default_profile: go-dev    # used when -p is omitted and only one profile exists

profiles:
  go-dev:
    docker:
      host: tcp://...      # overrides global docker.host for this profile only
      attach_host: ...     # host used in the printed attach URL only
    build: ...
    run:
      env: ...
      mounts: ...
      port:
        bind: 127.0.0.1
```

`config.Load` resolves the file in order: explicit `-c` path → `./opencode-sandbox.yaml` → `$XDG_CONFIG_HOME/opencode-sandbox/config.yaml`. It auto-selects the profile if only one exists. The loaded `Config.DockerHost` is resolved in priority order: CLI `--docker-host`/`-H` flag → profile `docker.host` → global `docker.host`.

### CLI layer (`internal/cli`)

`rootFlags` (config path, profile, docker host) are persistent flags threaded into all subcommands. `run` and `build` call `config.Load` then apply flag overrides (`dockerHost`, `--env`/`-e`, `--mount`/`-v`, `--bind`) before executing. `ps` and `rm` skip config loading and use `rf.dockerHost` directly with `docker.NewClient`.

### Build vs run split

- **`internal/build`** — shells out to `docker build` (uses BuildKit secrets), sets `DOCKER_HOST` from `cfg.DockerHost`
- **`internal/run`** — uses the Docker Go SDK to create/start the container, publish port 4096/tcp to a random host port, and return the assigned port; when `docker.macos: true` is set in config and the tool runs on macOS (`runtime.GOOS == "darwin"`), bind mount sources are validated against the local macOS filesystem before the API call to surface missing-path errors early
- **`internal/docker`** — thin wrapper: `NewClient` (SDK client), `EffectiveHost` (config → env fallback), `AttachHost` (derives printable hostname from a Docker host URL); `NewClient` resolves the daemon endpoint in the same order as the Docker CLI: explicit host → `DOCKER_HOST` env → active Docker context (`DOCKER_CONTEXT` env or `currentContext` in `~/.docker/config.json`) → SDK default

### Safety invariant

Every container created by the tool carries the label `opencode-sandbox=true` (constant in `internal/sandbox`). `ps` filters by this label; `rm` refuses to touch containers that don't have it.

### Examples (`examples/`)

After any change that affects config schema, CLI flags, or runtime behaviour, update the files under `examples/` to reflect it. The examples are the primary user-facing reference — keep their comments and field choices consistent with the current feature set.

### Path resolution (`internal/paths`)

`paths.Expand` handles `~`, `~/`, `$VAR`, and relative paths (resolved against the config file's directory, not CWD) uniformly across mount sources, Dockerfile paths, and build contexts.
