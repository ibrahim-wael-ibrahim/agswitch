package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateCommand(dependencies *dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "update <profile>",
		Short: "Replace a profile with the active credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if err := dependencies.app.Update(command.Context(), args[0]); err != nil {
				return err
			}
			_, err := fmt.Fprintf(command.OutOrStdout(), "Updated: %s\n", args[0])
			return err
		},
	}
}

func newCloneCommand(dependencies *dependencies) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "clone <source> <target>",
		Short: "Copy a saved profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			if err := dependencies.app.Clone(command.Context(), args[0], args[1], force); err != nil {
				return err
			}
			_, err := fmt.Fprintf(command.OutOrStdout(), "Cloned: %s -> %s\n", args[0], args[1])
			return err
		},
	}
	command.Flags().BoolVarP(&force, "force", "f", false, "overwrite the target profile")
	return command
}

func newRenameCommand(dependencies *dependencies) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "rename <source> <target>",
		Short: "Rename a saved profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			if err := dependencies.app.Rename(command.Context(), args[0], args[1], force); err != nil {
				return err
			}
			_, err := fmt.Fprintf(command.OutOrStdout(), "Renamed: %s -> %s\n", args[0], args[1])
			return err
		},
	}
	command.Flags().BoolVarP(&force, "force", "f", false, "overwrite the target profile")
	return command
}

func newInfoCommand(dependencies *dependencies) *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
		Use:   "info <profile>",
		Short: "Show non-sensitive profile details",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			info, err := dependencies.app.Info(command.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(command.OutOrStdout()).Encode(info)
			}
			item := info.Account
			fmt.Fprintf(command.OutOrStdout(), "Name: %s\n", item.ID)
			fmt.Fprintf(command.OutOrStdout(), "Email: %s\n", item.Email)
			fmt.Fprintf(command.OutOrStdout(), "Storage: %s\n", info.Storage)
			fmt.Fprintf(command.OutOrStdout(), "Credential type: %s\n", info.CredentialType)
			fmt.Fprintf(command.OutOrStdout(), "Created: %s\n", formatTime(item.CreatedAt))
			fmt.Fprintf(command.OutOrStdout(), "Updated: %s\n", formatTime(item.UpdatedAt))
			fmt.Fprintf(command.OutOrStdout(), "Last used: %s\n", formatTime(item.LastUsedAt))
			return nil
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return command
}

func newDetectCommand(dependencies *dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Detect whether the active credential is saved",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			item, found, err := dependencies.app.Current(command.Context())
			if err != nil {
				return err
			}
			if !found {
				_, err = fmt.Fprintln(command.OutOrStdout(), "Active credential is not saved. Use: agswitch save <profile>")
				return err
			}
			_, err = fmt.Fprintf(command.OutOrStdout(), "Active credential matches: %s (%s)\n", item.ID, item.Email)
			return err
		},
	}
}
