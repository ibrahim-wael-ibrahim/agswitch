package cmd

import (
	"context"
	"fmt"

	agsapp "github.com/ibrahim-wael/agswitch/internal/app"
	"github.com/spf13/cobra"
)

func newMigrateCommand(dependencies *dependencies) *cobra.Command {
	var source string
	var force bool
	var deleteSource bool
	command := &cobra.Command{
		Use:   "migrate",
		Short: "Import legacy JSON profiles",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			results, err := migrateProfiles(command.Context(), dependencies, source, force, deleteSource)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				_, err = fmt.Fprintln(command.OutOrStdout(), "No legacy profile files found.")
				return err
			}
			failed := false
			for _, result := range results {
				if result.Err != nil {
					failed = true
					fmt.Fprintf(command.ErrOrStderr(), "[FAIL] %s: %v\n", result.Profile, result.Err)
					continue
				}
				fmt.Fprintf(command.OutOrStdout(), "[OK] %s: %s\n", result.Profile, result.Status)
			}
			if failed {
				return fmt.Errorf("one or more profiles could not be migrated")
			}
			return nil
		},
	}
	command.Flags().StringVar(&source, "source", dependencies.config.BaseDir, "source directory")
	command.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing profiles")
	command.Flags().BoolVar(&deleteSource, "delete-source", false, "delete verified source files")
	return command
}

func migrateProfiles(ctx context.Context, dependencies *dependencies, source string, force, deleteSource bool) (results []agsapp.MigrationResult, err error) {
	service := *dependencies.app
	if service.Locker == nil {
		return service.Migrate(ctx, source, force, deleteSource)
	}
	unlock, err := service.Locker.Lock(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if unlockErr := unlock(); err == nil && unlockErr != nil {
			err = unlockErr
		}
	}()
	service.Locker = nil
	return service.Migrate(ctx, source, force, deleteSource)
}
