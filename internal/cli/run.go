package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/preved911/container-sandbox/internal/build"
	"github.com/preved911/container-sandbox/internal/config"
	"github.com/preved911/container-sandbox/internal/docker"
	"github.com/preved911/container-sandbox/internal/run"
)

func newRunCmd(rf *rootFlags) *cobra.Command {
	var (
		nameOverride string
		noBuild      bool
		pull         bool
	)
	cmd := &cobra.Command{
		Use:   "run [profile]",
		Short: "Build and start a sandbox, then print the opencode attach URL",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var profile string
			if len(args) == 1 {
				profile = args[0]
			}
			cfg, err := config.Load(rf.configPath, profile)
			if err != nil {
				return err
			}
			if nameOverride != "" {
				cfg.Name = nameOverride
			}
			if cfg.Name == "" {
				return fmt.Errorf("sandbox name is empty: set `name:` in the config or pass --name")
			}

			ctx := cmd.Context()

			var image string
			switch {
			case cfg.Build.Image != "":
				image = cfg.Build.Image
			case noBuild:
				image = "container-sandbox/" + cfg.Name + ":latest"
			default:
				image, err = build.ImageBuild(ctx, cfg, build.Options{Pull: pull})
				if err != nil {
					return err
				}
			}

			cli, err := docker.NewClient(cfg.Docker.Host)
			if err != nil {
				return fmt.Errorf("docker client: %w", err)
			}
			defer cli.Close()

			res, err := run.Start(ctx, cli, cfg, image, cfg.Name)
			if err != nil {
				return err
			}

			host := cfg.Docker.AttachHost
			if host == "" {
				host = docker.AttachHost(docker.EffectiveHost(cfg.Docker.Host))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "opencode attach http://%s:%d\n", host, res.HostPort)
			return nil
		},
	}
	cmd.Flags().StringVar(&nameOverride, "name", "", "override the sandbox/container name")
	cmd.Flags().BoolVar(&noBuild, "no-build", false, "skip the build step (image must already exist)")
	cmd.Flags().BoolVar(&pull, "pull", false, "pass --pull to docker build")
	return cmd
}
