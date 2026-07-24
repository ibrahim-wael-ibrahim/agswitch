package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newQuotaCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "quota [profile]",
		Short: "Show quota information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "quota view (bootstrap only)")
				return err
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "quota for %q (bootstrap only)\n", args[0])
			return err
		},
	}
}