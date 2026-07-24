package cmd

import (
	"github.com/ibrahim-wael/agswitch/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive terminal UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}
}
