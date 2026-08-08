package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/doctor"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/ibrahim-wael/agswitch/internal/tui"
	"github.com/spf13/cobra"
)

const defaultDashboardRefreshSeconds = 60

type dashboardBackend struct {
	dependencies *dependencies
}

func (b dashboardBackend) List(ctx context.Context) ([]account.Account, error) {
	return b.dependencies.app.List(ctx)
}

func (b dashboardBackend) Use(ctx context.Context, profile string, options switcher.Options) error {
	return b.dependencies.app.Use(ctx, profile, options)
}

func (b dashboardBackend) Update(ctx context.Context, profile string) error {
	return b.dependencies.app.Update(ctx, profile)
}

func (b dashboardBackend) FetchAll(ctx context.Context, accounts []account.Account, refresh bool) []quota.Result {
	return b.dependencies.quota.FetchAll(ctx, accounts, refresh)
}

func (b dashboardBackend) Previous(ctx context.Context) (string, error) {
	snapshot, err := b.dependencies.state.Load(ctx)
	if err != nil {
		return "", err
	}
	if snapshot.Previous == "" {
		return "", fmt.Errorf("no previous profile is recorded")
	}
	if err := b.dependencies.app.Use(ctx, snapshot.Previous, switcher.Options{LaunchMode: switcher.AlwaysLaunch}); err != nil {
		return "", err
	}
	return snapshot.Previous, nil
}

func (b dashboardBackend) Doctor(ctx context.Context) []doctor.Check {
	return b.dependencies.doctor.Run(ctx)
}

func newTUICommand(dependencies *dependencies) *cobra.Command {
	var stay bool
	var autoThreshold int
	var autoRefreshSeconds int
	command := &cobra.Command{
		Use:   "tui",
		Short: "Open the responsive interactive dashboard",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if autoRefreshSeconds < 0 {
				return fmt.Errorf("--auto-refresh must be 0 or a positive number of seconds")
			}
			return runTUIWithOptions(command, dependencies, stay, autoThreshold, autoRefreshSeconds)
		},
	}
	command.Flags().BoolVar(&stay, "stay", true, "stay open after switching")
	command.Flags().IntVar(&autoThreshold, "auto-threshold", 20, "recommend auto-switch when minimum known quota is at or below this percentage")
	command.Flags().IntVar(&autoRefreshSeconds, "auto-refresh", defaultDashboardRefreshSeconds, "refresh live quota for all saved accounts every N seconds; 0 disables automatic refresh")
	return command
}

func runTUI(command *cobra.Command, dependencies *dependencies, stay bool) error {
	return runTUIWithOptions(command, dependencies, stay, 20, defaultDashboardRefreshSeconds)
}

func runTUIWithOptions(command *cobra.Command, dependencies *dependencies, stay bool, autoThreshold, autoRefreshSeconds int) error {
	return tui.RunDashboard(command.Context(), dashboardBackend{dependencies: dependencies}, tui.Options{
		Version:         resolvedVersion(),
		AutoThreshold:   autoThreshold,
		AutoRefresh:     time.Duration(autoRefreshSeconds) * time.Second,
		ExitAfterSwitch: !stay,
	})
}
