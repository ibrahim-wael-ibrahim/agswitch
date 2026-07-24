package cmd

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
)

var version = "dev"

func newStatusCommand(dependencies *dependencies) *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
		Use:   "status",
		Short: "Show account and Antigravity status",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			item, matched, err := dependencies.app.Current(command.Context())
			if err != nil {
				return err
			}
			running, err := dependencies.process.Running(command.Context())
			if err != nil {
				return err
			}
			stateSnapshot, err := dependencies.state.Load(command.Context())
			if err != nil {
				return err
			}
			output := map[string]any{
				"profile":             item.ID,
				"email":               item.Email,
				"credential_matches":  matched,
				"application_running": running,
				"previous":            stateSnapshot.Previous,
				"updated_at":          stateSnapshot.UpdatedAt,
			}
			if asJSON {
				return json.NewEncoder(command.OutOrStdout()).Encode(output)
			}
			profileName := "unknown"
			if matched {
				profileName = item.ID
			}
			fmt.Fprintf(command.OutOrStdout(), "Current profile: %s\n", profileName)
			fmt.Fprintf(command.OutOrStdout(), "Credential: %s\n", map[bool]string{true: "matched", false: "not saved"}[matched])
			fmt.Fprintf(command.OutOrStdout(), "Antigravity: %s\n", map[bool]string{true: "running", false: "stopped"}[running])
			if stateSnapshot.Previous != "" {
				fmt.Fprintf(command.OutOrStdout(), "Previous profile: %s\n", stateSnapshot.Previous)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return command
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			resolved := version
			if resolved == "dev" {
				if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
					resolved = info.Main.Version
				}
			}
			_, err := fmt.Fprintln(command.OutOrStdout(), resolved)
			return err
		},
	}
}
