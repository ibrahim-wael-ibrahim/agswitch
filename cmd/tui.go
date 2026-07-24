package cmd

import (
	"context"
	"fmt"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/doctor"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/ibrahim-wael/agswitch/internal/tui"
	"github.com/spf13/cobra"
)

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
	command := &cobra.Command{
		Use:   "tui",
		Short: "Open the responsive interactive dashboard",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runTUIWithThreshold(command, dependencies, stay, autoThreshold)
		},
	}
	command.Flags().BoolVar(&stay, "stay", true, "stay open after switching")
	command.Flags().IntVar(&autoThreshold, "auto-threshold", 20, "recommend auto-switch when minimum known quota is at or below this percentage")
	return command
}

func runTUI(command *cobra.Command, dependencies *dependencies, stay bool) error {
	return runTUIWithThreshold(command, dependencies, stay, 20)
}

func runTUIWithThreshold(command *cobra.Command, dependencies *dependencies, stay bool, autoThreshold int) error {
	return tui.Run(command.Context(), dashboardBackend{dependencies: dependencies}, tui.Options{
		Version:         resolvedVersion(),
		AutoThreshold:   autoThreshold,
		ExitAfterSwitch: !stay,
	})
}
