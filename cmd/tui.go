package cmd

import (
	"github.com/ibrahim-wael/agswitch/internal/fzfui"
	"github.com/spf13/cobra"
)

func newTUICommand(dependencies *dependencies) *cobra.Command {
	var stay bool
	command := &cobra.Command{
		Use:   "tui",
		Short: "Open the fzf quota dashboard",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runTUI(command, dependencies, stay)
		},
	}
	command.Flags().BoolVar(&stay, "stay", false, "stay open after switching")
	return command
}

func runTUI(command *cobra.Command, dependencies *dependencies, stay bool) error {
	return fzfui.Run(command.Context(), dependencies.app, dependencies.quota, fzfui.Options{
		Stay:   stay,
		Stdin:  command.InOrStdin(),
		Stdout: command.OutOrStdout(),
		Stderr: command.ErrOrStderr(),
	})
}
