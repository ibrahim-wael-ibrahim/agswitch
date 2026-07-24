package tui

import tea "charm.land/bubbletea/v2"

func (m Model) Update(x tea.Msg) (tea.Model, tea.Cmd) {
	switch v := x.(type) {
	case tea.KeyMsg:
		if m.Switching {
			return m, nil
		}
		switch v.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.Selected > 0 {
				m.Selected--
			}
		case "down", "j":
			if m.Selected+1 < len(m.Accounts) {
				m.Selected++
			}
		case "r":
			a, e := m.Backend.List(m.Context)
			if e != nil {
				m.Status = e.Error()
			} else {
				m.SetAccounts(a)
				m.Status = "Refreshed"
			}
		case "enter":
			a, ok := m.SelectedAccount()
			if ok {
				m.Switching = true
				m.Status = "Switching to " + a.ID + "..."
				return m, m.switchCommand(a.ID)
			}
		}
	case switchResultMsg:
		m.Switching = false
		if v.err != nil {
			m.Status = "Switch failed: " + v.err.Error()
			return m, nil
		}
		for i := range m.Accounts {
			m.Accounts[i].Active = m.Accounts[i].ID == v.profile
		}
		m.Status = "Started Antigravity with " + v.profile
		if !m.Stay {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = v.Width
		m.Height = v.Height
	}
	return m, nil
}
