package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/ibrahim-wael/agswitch/internal/brand"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

var version = "v1.0.0"

func resolvedVersion() string {
	resolved := version
	if resolved == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			resolved = info.Main.Version
		}
	}
	return resolved
}

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

func newPreviousCommand(dependencies *dependencies) *cobra.Command {
	var restart bool
	var noStart bool
	command := &cobra.Command{
		Use:     "previous",
		Aliases: []string{"prev"},
		Short:   "Switch to the previously active profile",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if restart && noStart {
				return fmt.Errorf("--restart and --no-start cannot be used together")
			}
			snapshot, err := dependencies.state.Load(command.Context())
			if err != nil {
				return err
			}
			if snapshot.Previous == "" {
				return fmt.Errorf("no previous profile is recorded")
			}
			mode := switcher.PreserveLaunchState
			if restart {
				mode = switcher.AlwaysLaunch
			} else if noStart {
				mode = switcher.NeverLaunch
			}
			if err := dependencies.app.Use(command.Context(), snapshot.Previous, switcher.Options{LaunchMode: mode}); err != nil {
				return err
			}
			_, err = fmt.Fprintf(command.OutOrStdout(), "Done: %s\n", snapshot.Previous)
			return err
		},
	}
	command.Flags().BoolVar(&restart, "restart", false, "always launch Antigravity")
	command.Flags().BoolVar(&noStart, "no-start", false, "leave Antigravity stopped")
	return command
}

func newVersionCommand() *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
		Use:   "version",
		Short: "Print version and project information",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			resolved := resolvedVersion()
			if asJSON {
				return json.NewEncoder(command.OutOrStdout()).Encode(map[string]string{
					"name":        brand.Name,
					"version":     resolved,
					"author":      brand.Author,
					"github_user": brand.GitHubUser,
					"repository":  brand.Repository,
					"go_version":  runtime.Version(),
				})
			}
			fmt.Fprint(command.OutOrStdout(), brand.Banner(resolved))
			fmt.Fprintf(command.OutOrStdout(), "Runtime: %s · %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return command
}
