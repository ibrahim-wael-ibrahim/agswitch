package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newConfigCommand(d *dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Display the resolved configuration",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			config := d.config
			_, err := fmt.Fprintf(
				command.OutOrStdout(),
				"base_dir=%s\nstate_dir=%s\napp_path=%s\nlanguage_server_path=%s\nlog_path=%s\nquit_command=%s\ngraceful_timeout=%s\nforce_kill=%t\n",
				config.BaseDir,
				config.StateDir,
				config.AppPath,
				config.LanguageServerPath,
				config.LogPath,
				strings.Join(config.QuitCommand, " "),
				config.GracefulTimeout,
				config.ForceKill,
			)
			return err
		},
	}
}
