package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func newSaveCommand(d *dependencies) *cobra.Command {
	var f bool
	c := &cobra.Command{Use: "save <profile>", Short: "Save active credential", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, a []string) error {
		if e := d.app.Save(c.Context(), a[0], f); e != nil {
			return e
		}
		_, e := fmt.Fprintf(c.OutOrStdout(), "Saved: %s\n", a[0])
		return e
	}}
	c.Flags().BoolVarP(&f, "force", "f", false, "overwrite")
	return c
}
