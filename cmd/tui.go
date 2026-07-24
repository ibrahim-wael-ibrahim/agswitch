package cmd

import (
	"github.com/ibrahim-wael/agswitch/internal/fzfui"
	"github.com/spf13/cobra"
)

func newTUICommand(dependencies *dependencies) *cobra.Command {
	var stay bool
	var autoThreshold int
	command := &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive quota dashboard",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runTUIWithThreshold(command, dependencies, stay, autoThreshold)
		},
	}
	command.Flags().BoolVar(&stay, "stay", false, "stay open after switching")
	command.Flags().IntVar(&autoThreshold, "auto-threshold", 20, "recommend auto-switch when minimum known quota is at or below this percentage")
	return command
}

func runTUI(command *cobra.Command, dependencies *dependencies, stay bool) error {
	return runTUIWithThreshold(command, dependencies, stay, 20)
}

func runTUIWithThreshold(command *cobra.Command, dependencies *dependencies, stay bool, autoThreshold int) error {
	return fzfui.Run(command.Context(), dependencies.app, dependencies.quota, fzfui.Options{
		Stay:          stay,
		Version:       resolvedVersion(),
		AutoThreshold: autoThreshold,
		Stdin:         command.InOrStdin(),
		Stdout:        command.OutOrStdout(),
		Stderr:        command.ErrOrStderr(),
	})
}
