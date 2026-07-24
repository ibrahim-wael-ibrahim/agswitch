package cmd

import (
	"fmt"

	"github.com/ibrahim-wael/agswitch/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Display the resolved configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Default()
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "base_dir=%s\nstate_dir=%s\napp_path=%s\n", cfg.BaseDir, cfg.StateDir, cfg.AppPath)
			return err
		},
	}
}