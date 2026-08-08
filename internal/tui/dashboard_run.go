package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// RunDashboard starts the quota-first responsive dashboard. It preserves the
// existing Model behavior while using the newer presentation layer.
func RunDashboard(ctx context.Context, backend Backend, options Options) error {
	if backend == nil {
		return fmt.Errorf("dashboard backend is not configured")
	}
	program := tea.NewProgram(newDashboardModel(New(ctx, backend, options)))
	_, err := program.Run()
	return err
}
