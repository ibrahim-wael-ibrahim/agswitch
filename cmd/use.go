package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile>",
		Short: "Switch to a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "use profile %q (bootstrap only)\n", args[0])
			return err
		},
	}
}