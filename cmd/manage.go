package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateCommand(d *dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "update <profile>",
		Short: "Replace a saved profile with the active credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if err := d.app.Update(c.Context(), args[0]); err != nil {
				return err
			}
			_, err := fmt.Fprintf(c.OutOrStdout(), "Updated: %s\n", args[0])
			return err
		},
	}
}

func newCloneCommand(d *dependencies) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "clone <source> <target>",
		Short: "Copy a saved profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			if err := d.app.Clone(c.Context(), args[0], args[1], force); err != nil {
				return err
			}
			_, err := fmt.Fprintf(c.OutOrStdout(), "Cloned: %s -> %s\n", args[0], args[1])
			return err
		},
	}
	command.Flags().BoolVarP(&force, "force", "f", false, "overwrite the target profile")
	return command
}

func newRenameCommand(d *dependencies) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "rename <source> <target>",
		Short: "Rename a saved profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			if err := d.app.Rename(c.Context(), args[0], args[1], force); err != nil {
				return err
			}
			_, err := fmt.Fprintf(c.OutOrStdout(), "Renamed: %s -> %s\n", args[0], args[1])
			return err
		},
	}
	command.Flags().BoolVarP(&force, "force", "f", false, "overwrite the target profile")
	return command
}

func newInfoCommand(d *dependencies) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "info <profile>",
		Short: "Show non-secret profile details",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			info, err := d.app.Info(c.Context(), args[0])
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(c.OutOrStdout()).Encode(info)
			}
			account := info.Account
			_, err = fmt.Fprintf(c.OutOrStdout(), "Name: %s\nEmail: %s\nStorage: %s\nCredential type: %s\nCreated: %s\nUpdated: %s\nLast used: %s\n",
				account.ID, account.Email, info.Storage, info.CredentialType,
				formatTimestamp(account.CreatedAt), formatTimestamp(account.UpdatedAt), formatTimestamp(account.LastUsedAt))
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON")
	return command
}

func formatTimestamp(value interface{ IsZero() bool; Format(string) string }) string {
	if value.IsZero() {
		return "never"
	}
	return value.Format("2006-01-02 15:04:05 MST")
}
