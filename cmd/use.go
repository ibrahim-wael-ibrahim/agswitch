package cmd

import (
	"fmt"

	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

func newUseCommand(d *dependencies) *cobra.Command {
	var restart, noStart, hotReload, confirmIdle bool
	command := &cobra.Command{
		Use:   "use <profile>",
		Short: "Switch transactionally",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if restart && noStart {
				return fmt.Errorf("--restart and --no-start cannot be used together")
			}
			if hotReload && (restart || noStart) {
				return fmt.Errorf("--hot-reload cannot be combined with --restart or --no-start")
			}
			if hotReload && !confirmIdle {
				return fmt.Errorf("hot reload requires --confirm-idle after confirming the current task and tool calls have finished")
			}
			mode := switcher.PreserveLaunchState
			if restart {
				mode = switcher.AlwaysLaunch
			}
			if noStart {
				mode = switcher.NeverLaunch
			}
			options := switcher.Options{LaunchMode: mode, HotReload: hotReload}
			if err := d.app.Use(command.Context(), args[0], options); err != nil {
				return err
			}
			message := "Done"
			if hotReload {
				message = "Hot-switched language server"
			}
			_, err := fmt.Fprintf(command.OutOrStdout(), "%s: %s\n", message, args[0])
			return err
		},
	}
	command.Flags().BoolVar(&restart, "restart", false, "always launch Antigravity, using a full application restart when already running")
	command.Flags().BoolVar(&noStart, "no-start", false, "leave Antigravity stopped")
	command.Flags().BoolVar(&hotReload, "hot-reload", false, "keep the Electron UI open and restart only the Antigravity language server")
	command.Flags().BoolVar(&confirmIdle, "confirm-idle", false, "confirm that the current response and tool calls have finished")
	return command
}
