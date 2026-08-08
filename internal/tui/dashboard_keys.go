package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// friendlyDashboardModel adds direct account actions on top of dashboardModel.
// The quota-first layout intentionally has no hidden Actions panel, so Enter
// and the documented shortcuts always act on the visible selected account.
type friendlyDashboardModel struct {
	dashboardModel
}

func newFriendlyDashboardModel(model Model) friendlyDashboardModel {
	return friendlyDashboardModel{dashboardModel: newDashboardModel(model)}
}

func (m friendlyDashboardModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := message.(tea.KeyMsg); ok && !m.Busy && !m.Searching && !m.EditingRefresh && !m.EditingThreshold && !m.confirming() {
		switch key.String() {
		case "enter", "s":
			profile, ok := m.selectedProfile()
			if !ok {
				m.Status = "Select an account first"
				return m, nil
			}
			updated, cmd := m.beginConfirmation(actionHotReload, profile)
			m.Model = updated.(Model)
			return m, cmd
		case "f":
			profile, ok := m.selectedProfile()
			if !ok {
				m.Status = "Select an account first"
				return m, nil
			}
			m.Busy = true
			m.Status = "Restarting Antigravity with " + profile + "..."
			return m, m.operationCommand(actionSwitchLaunch)
		case "u":
			profile, ok := m.selectedProfile()
			if !ok {
				m.Status = "Select an account first"
				return m, nil
			}
			m.Busy = true
			m.Status = "Syncing renewed credential into " + profile + "..."
			return m, m.operationCommand(actionUpdate)
		case "o":
			profile, ok := m.selectedProfile()
			if !ok {
				m.Status = "Select an account first"
				return m, nil
			}
			m.Busy = true
			m.Status = "Activating " + profile + " without restarting Antigravity..."
			return m, m.operationCommand(actionSwitchOnly)
		case "tab", "shift+tab", "left", "right", "h", "l":
			// There is no hidden command panel in the quota-first dashboard.
			m.Focus = focusAccounts
			return m, nil
		}
	}

	updated, cmd := m.dashboardModel.Update(message)
	switch value := updated.(type) {
	case dashboardModel:
		m.dashboardModel = value
		return m, cmd
	case friendlyDashboardModel:
		return value, cmd
	default:
		return updated, cmd
	}
}

func (m friendlyDashboardModel) View() tea.View {
	view := m.dashboardModel.View()
	// Keep help truthful for the direct-action dashboard.
	if !m.confirming() && !m.Searching && !m.EditingRefresh && !m.EditingThreshold {
		view.Content = strings.ReplaceAll(view.Content,
			"↑/↓ account  Enter actions  s hot-switch  r refresh  a auto  / search  m models  Tab panels  q quit",
			"↑/↓ account  Enter/s hot-switch  f full restart  u sync  o activate  r refresh  a auto  / search  m models  q quit",
		)
	}
	return view
}
