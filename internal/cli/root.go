// Package cli wires the cobra command tree.
package cli

import (
	"context"

	"github.com/spf13/cobra"
)

// Execute runs the CLI with the given context (which should be signal-aware).
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}

type rootFlags struct {
	configPath string
	profile    string
}

func newRootCmd() *cobra.Command {
	rf := &rootFlags{}
	cmd := &cobra.Command{
		Use:           "container-sandbox",
		Short:         "Manage isolated opencode containers",
		Long:          "container-sandbox builds and runs containers that expose an opencode `serve` endpoint on a random host port, so you can attach a local opencode client to a sandboxed run.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	cmd.PersistentFlags().StringVarP(&rf.configPath, "config", "c", "", "config file path (default: ./container-sandbox.yaml → $HOME/.config/opencode-sandbox/config.yaml)")
	cmd.PersistentFlags().StringVarP(&rf.profile, "profile", "p", "", "profile name to load from a profiles config file")

	cmd.AddCommand(
		newRunCmd(rf),
		newBuildCmd(rf),
		newPsCmd(),
		newRmCmd(),
	)
	return cmd
}
