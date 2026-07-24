package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

func newUpdateCommand(d *dependencies) *cobra.Command {
	return &cobra.Command{Use: "update <profile>", Short: "Replace a saved profile with the active credential", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if err := d.app.Update(c.Context(), args[0]); err != nil { return err }
		_, err := fmt.Fprintf(c.OutOrStdout(), "Updated: %s\n", args[0]); return err
	}}
}

func newCloneCommand(d *dependencies) *cobra.Command {
	var force bool
	command := &cobra.Command{Use: "clone <source> <target>", Short: "Copy a saved profile", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		if err := d.app.Clone(c.Context(), args[0], args[1], force); err != nil { return err }
		_, err := fmt.Fprintf(c.OutOrStdout(), "Cloned: %s -> %s\n", args[0], args[1]); return err
	}}
	command.Flags().BoolVarP(&force, "force", "f", false, "overwrite target profile")
	return command
}

func newRenameCommand(d *dependencies) *cobra.Command {
	var force bool
	command := &cobra.Command{Use: "rename <source> <target>", Short: "Rename a saved profile transactionally", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		if err := d.app.Rename(c.Context(), args[0], args[1], force); err != nil { return err }
		_, err := fmt.Fprintf(c.OutOrStdout(), "Renamed: %s -> %s\n", args[0], args[1]); return err
	}}
	command.Flags().BoolVarP(&force, "force", "f", false, "overwrite target profile")
	return command
}

func newInfoCommand(d *dependencies) *cobra.Command {
	var outputJSON bool
	command := &cobra.Command{Use: "info <profile>", Short: "Show non-secret profile details", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		info, err := d.app.Info(c.Context(), args[0]); if err != nil { return err }
		if outputJSON { return json.NewEncoder(c.OutOrStdout()).Encode(info) }
		item := info.Account
		fmt.Fprintf(c.OutOrStdout(), "Name: %s\nEmail: %s\nStorage: %s\nCredential type: %s\nCreated: %s\nUpdated: %s\nLast used: %s\n", item.ID, item.Email, info.Storage, info.CredentialType, formatTimestamp(item.CreatedAt), formatTimestamp(item.UpdatedAt), formatTimestamp(item.LastUsedAt))
		return nil
	}}
	command.Flags().BoolVar(&outputJSON, "json", false, "print JSON")
	return command
}

func newDetectCommand(d *dependencies) *cobra.Command {
	var outputJSON bool
	command := &cobra.Command{Use: "detect", Short: "Match the active credential to a saved profile", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		item, matched, err := d.app.Current(c.Context()); if err != nil { return err }
		if outputJSON { return json.NewEncoder(c.OutOrStdout()).Encode(map[string]any{"matched": matched, "profile": item.ID, "email": item.Email}) }
		if !matched { _, err = fmt.Fprintln(c.OutOrStdout(), "Active credential is not saved. Use: agswitch save <profile>"); return err }
		_, err = fmt.Fprintf(c.OutOrStdout(), "Active credential matches: %s (%s)\n", item.ID, item.Email); return err
	}}
	command.Flags().BoolVar(&outputJSON, "json", false, "print JSON")
	return command
}

func newPreviousCommand(d *dependencies) *cobra.Command {
	var restart, noStart bool
	command := &cobra.Command{Use: "previous", Aliases: []string{"prev"}, Short: "Switch to the previously active profile", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		if restart && noStart { return fmt.Errorf("--restart and --no-start cannot be used together") }
		snapshot, err := d.state.Load(c.Context()); if err != nil { return err }
		if snapshot.Previous == "" { return fmt.Errorf("no previous profile is recorded") }
		mode := switcher.PreserveLaunchState; if restart { mode = switcher.AlwaysLaunch }; if noStart { mode = switcher.NeverLaunch }
		if err := d.app.Use(c.Context(), snapshot.Previous, switcher.Options{LaunchMode: mode}); err != nil { return err }
		_, err = fmt.Fprintf(c.OutOrStdout(), "Done: %s\n", snapshot.Previous); return err
	}}
	command.Flags().BoolVar(&restart, "restart", false, "always launch Antigravity")
	command.Flags().BoolVar(&noStart, "no-start", false, "leave Antigravity stopped")
	return command
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() { return "never" }
	return value.Local().Format("2006-01-02 15:04:05 MST")
}
