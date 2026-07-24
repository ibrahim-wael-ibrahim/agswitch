package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:          "agswitch",
	Short:        "Antigravity account switcher",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(
		newTUICommand(),
		newUseCommand(),
		newSaveCommand(),
		newListCommand(),
		newDeleteCommand(),
		newQuotaCommand(),
		newDoctorCommand(),
		newConfigCommand(),
	)
}
