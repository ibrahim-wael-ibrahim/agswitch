package cmd

import (
	"github.com/ibrahim-wael/agswitch/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICommand(d *dependencies) *cobra.Command {
	var stay bool
	c := &cobra.Command{Use: "tui", Short: "Launch terminal UI", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error { return runTUI(c, d, stay) }}
	c.Flags().BoolVar(&stay, "stay", false, "stay open")
	return c
}
func runTUI(c *cobra.Command, d *dependencies, stay bool) error {
	return tui.Run(c.Context(), d.app, stay)
}
