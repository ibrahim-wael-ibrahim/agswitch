package tui

import tea "charm.land/bubbletea/v2"

func (m Model) View() tea.View {
	if len(m.Accounts) == 0 {
		return tea.NewView("agswitch\n\nNo accounts loaded yet.\n\nPress q to quit.\n")
	}

	selected := m.Selected
	if selected < 0 || selected >= len(m.Accounts) {
		selected = 0
	}

	account := m.Accounts[selected]
	return tea.NewView("agswitch\n\nSelected profile: " + account.ID + "\nStatus: " + m.Status + "\n\nPress q to quit.\n")
}
