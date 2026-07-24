package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	case dataMsg:
		m.Busy = false
		if msg.err != nil {
			m.Status = "Load failed: " + msg.err.Error()
			return m, nil
		}
		m.Accounts = msg.accounts
		m.Results = msg.results
		m.Decision = msg.decision
		m.clampSelection()
		if msg.refresh {
			m.Status = "Live quota refreshed"
		} else {
			m.Status = "Ready"
		}
		return m, nil
	case operationMsg:
		m.Busy = false
		if msg.err != nil {
			m.Status = "Command failed: " + msg.err.Error()
			return m, nil
		}
		if msg.details != "" {
			m.Details = msg.details
		}
		switch msg.action {
		case actionDoctor:
			m.Status = "Doctor completed"
			return m, nil
		case actionRefresh:
			m.Busy = true
			m.Status = "Refreshing live quota..."
			return m, m.loadDataCommand(true)
		case actionQuit:
			return m, tea.Quit
		default:
			if msg.profile != "" {
				m.Status = fmt.Sprintf("Completed %s for %s", msg.action, msg.profile)
			} else {
				m.Status = "Command completed"
			}
			if m.Options.ExitAfterSwitch && (msg.action == actionSwitchLaunch || msg.action == actionAutoSwitch) {
				return m, tea.Quit
			}
			m.Busy = true
			return m, m.loadDataCommand(false)
		}
	case tea.KeyMsg:
		if m.Busy {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}
		if m.Searching {
			return m.updateSearch(msg)
		}
		return m.updateNavigation(msg)
	}
	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Searching = false
		m.Focus = focusAccounts
		return m, nil
	case "enter":
		m.Searching = false
		m.Focus = focusAccounts
		m.SelectedAccount = 0
		m.Status = fmt.Sprintf("Filter: %q", m.Search)
		return m, nil
	case "backspace":
		if m.Search != "" {
			_, size := utf8.DecodeLastRuneInString(m.Search)
			m.Search = m.Search[:len(m.Search)-size]
			m.SelectedAccount = 0
		}
		return m, nil
	case "ctrl+u":
		m.Search = ""
		m.SelectedAccount = 0
		return m, nil
	}
	text := msg.String()
	if len(text) == 1 && text >= " " && text != "\x7f" {
		m.Search += text
		m.SelectedAccount = 0
	}
	return m, nil
}

func (m Model) updateNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab", "right", "l":
		if m.Focus == focusCommands {
			m.Focus = focusAccounts
		} else {
			m.Focus = focusCommands
		}
	case "shift+tab", "left", "h":
		if m.Focus == focusAccounts {
			m.Focus = focusCommands
		} else {
			m.Focus = focusAccounts
		}
	case "/":
		m.Searching = true
		m.Focus = focusSearch
		m.Status = "Type to filter accounts; Enter applies, Esc cancels"
	case "up", "k":
		if m.Focus == focusCommands {
			if m.SelectedCommand > 0 {
				m.SelectedCommand--
			}
		} else if m.SelectedAccount > 0 {
			m.SelectedAccount--
		}
	case "down", "j":
		if m.Focus == focusCommands {
			if m.SelectedCommand+1 < len(dashboardCommands) {
				m.SelectedCommand++
			}
		} else if m.SelectedAccount+1 < len(m.filteredAccounts()) {
			m.SelectedAccount++
		}
	case "enter":
		if m.Focus == focusCommands {
			selected := dashboardCommands[m.SelectedCommand].Action
			if selected == actionQuit {
				return m, tea.Quit
			}
			if selected == actionRefresh {
				m.Busy = true
				m.Status = "Refreshing live quota..."
				return m, m.loadDataCommand(true)
			}
			m.Busy = true
			m.Status = "Running " + strings.ToLower(dashboardCommands[m.SelectedCommand].Label) + "..."
			return m, m.operationCommand(selected)
		}
		m.Focus = focusCommands
		m.SelectedCommand = 0
		m.Status = "Account selected; choose a command and press Enter"
	case "r":
		m.Busy = true
		m.Status = "Refreshing live quota..."
		return m, m.loadDataCommand(true)
	case "a":
		m.Busy = true
		m.Status = "Applying auto-switch recommendation..."
		return m, m.operationCommand(actionAutoSwitch)
	case "p":
		m.Busy = true
		m.Status = "Switching to previous account..."
		return m, m.operationCommand(actionPrevious)
	case "d":
		m.Busy = true
		m.Status = "Running doctor..."
		return m, m.operationCommand(actionDoctor)
	}
	return m, nil
}

func (m *Model) clampSelection() {
	if m.SelectedCommand < 0 || m.SelectedCommand >= len(dashboardCommands) {
		m.SelectedCommand = 0
	}
	count := len(m.filteredAccounts())
	if count == 0 {
		m.SelectedAccount = 0
		return
	}
	if m.SelectedAccount < 0 || m.SelectedAccount >= count {
		m.SelectedAccount = 0
	}
}
