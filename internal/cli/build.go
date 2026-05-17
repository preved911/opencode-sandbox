package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/preved911/container-sandbox/internal/build"
	"github.com/preved911/container-sandbox/internal/config"
)

func newBuildCmd(rf *rootFlags) *cobra.Command {
	var pull bool
	cmd := &cobra.Command{
		Use:   "build [profile]",
		Short: "Build the sandbox image without starting a container",
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
			if cfg.Build.Image != "" {
				return fmt.Errorf("config has build.image set; nothing to build")
			}
			tag, err := build.ImageBuild(cmd.Context(), cfg, build.Options{Pull: pull})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), tag)
			return nil
		},
	}
	cmd.Flags().BoolVar(&pull, "pull", false, "pass --pull to docker build")
	return cmd
}
