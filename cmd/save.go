package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSaveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "save <profile>",
		Short: "Save the active credential as a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "save profile %q (bootstrap only)\n", args[0])
			return err
		},
	}
}