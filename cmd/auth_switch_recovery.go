package cmd

import (
	"fmt"

	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

// useProfileWithRecovery prefers the UI-preserving backend reload. If the
// language-server generation cannot be replaced, recover by doing a full
// Antigravity restart with the requested profile. refresh-via-antigravity is
// explicitly idle-confirmed, so this fallback is safer than leaving Electron
// alive without a backend.
func useProfileWithRecovery(command *cobra.Command, d *dependencies, profile, action string, quiet bool) error {
	hotErr := d.app.Use(command.Context(), profile, switcher.Options{
		LaunchMode: switcher.PreserveLaunchState,
		HotReload:  true,
	})
	if hotErr == nil {
		return nil
	}
	if !quiet {
		fmt.Fprintf(command.OutOrStdout(), "[RECOVER] %s hot reload failed; full restart fallback\n", action)
	}
	fullErr := d.app.Use(command.Context(), profile, switcher.Options{
		LaunchMode: switcher.AlwaysLaunch,
	})
	if fullErr == nil {
		return nil
	}
	return fmt.Errorf("hot reload failed: %v; full restart fallback failed: %w", hotErr, fullErr)
}
