package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ibrahim-wael/agswitch/internal/autoswitch"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

func newAutoSwitchCommand(dependencies *dependencies) *cobra.Command {
	var threshold int
	var refresh bool
	var dryRun bool
	var forceRunning bool
	var hotReload bool
	var confirmIdle bool
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "auto-switch",
		Short: "Select the safest account from recent live quota",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if hotReload && forceRunning {
				return fmt.Errorf("--hot-reload and --force-running select different restart strategies")
			}
			if hotReload && !confirmIdle {
				return fmt.Errorf("hot reload requires --confirm-idle after confirming the current task and tool calls have finished")
			}
			accounts, err := dependencies.app.List(command.Context())
			if err != nil {
				return err
			}
			if len(accounts) == 0 {
				return fmt.Errorf("no saved profiles")
			}
			current, matched, err := dependencies.app.Current(command.Context())
			if err != nil {
				return err
			}
			currentProfile := ""
			if matched {
				currentProfile = current.ID
			}
			results := dependencies.quota.FetchAll(command.Context(), accounts, refresh)
			decision := autoswitch.Select(results, currentProfile, threshold)
			if jsonOutput {
				if err := json.NewEncoder(command.OutOrStdout()).Encode(decision); err != nil {
					return err
				}
			} else {
				printAutoSwitchDecision(command, decision)
			}
			if !decision.Switch || dryRun {
				return nil
			}
			running, err := dependencies.process.Running(command.Context())
			if err != nil {
				return err
			}
			if running && !forceRunning && !hotReload {
				return fmt.Errorf("Antigravity is running; use --hot-reload --confirm-idle, or --force-running for a full restart")
			}
			options := switcher.Options{LaunchMode: switcher.PreserveLaunchState}
			if running && hotReload {
				options.HotReload = true
			} else if running {
				options.LaunchMode = switcher.AlwaysLaunch
			}
			if err := dependencies.app.Use(command.Context(), decision.Selected.Profile, options); err != nil {
				return err
			}
			if !jsonOutput {
				fmt.Fprintf(command.OutOrStdout(), "Switched to %s\n", decision.Selected.Profile)
			}
			return nil
		},
	}
	command.Flags().IntVar(&threshold, "threshold", 20, "switch when the current profile's minimum recent live quota is at or below this percentage")
	command.Flags().BoolVarP(&refresh, "refresh", "r", false, "bypass quota cache")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show the decision without switching")
	command.Flags().BoolVar(&forceRunning, "force-running", false, "allow a full application restart while Antigravity is running")
	command.Flags().BoolVar(&hotReload, "hot-reload", false, "keep the Electron UI open and restart only the language server")
	command.Flags().BoolVar(&confirmIdle, "confirm-idle", false, "confirm that the current response and tool calls have finished")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON")
	return command
}

func printAutoSwitchDecision(command *cobra.Command, decision autoswitch.Decision) {
	fmt.Fprintf(command.OutOrStdout(), "Auto-switch threshold: %d%%\n", decision.Threshold)
	if decision.Current.Profile != "" {
		fmt.Fprintf(command.OutOrStdout(), "Current:  %s (%d%% minimum recent live quota)\n", decision.Current.Profile, decision.Current.MinimumRemaining)
	} else {
		fmt.Fprintln(command.OutOrStdout(), "Current:  unknown or missing recent live quota")
	}
	if decision.Selected.Profile != "" {
		fmt.Fprintf(command.OutOrStdout(), "Selected: %s (%d%% minimum recent live quota)\n", decision.Selected.Profile, decision.Selected.MinimumRemaining)
	}
	if decision.Excluded > 0 {
		fmt.Fprintf(command.OutOrStdout(), "Excluded: %d profiles with failed, stale, old, or unknown quota\n", decision.Excluded)
	}
	fmt.Fprintf(command.OutOrStdout(), "Decision: %s\n", decision.Reason)
}
