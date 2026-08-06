package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
			return m, m.scheduleAutoRefresh()
		}
		firstLoad := !m.Initialized
		m.Accounts = msg.accounts
		m.Results = msg.results
		m.Decision = msg.decision
		if firstLoad {
			for index, item := range m.filteredAccounts() {
				if item.Active {
					m.SelectedAccount = index
					break
				}
			}
			m.Initialized = true
		}
		m.clampSelection()
		if msg.refresh {
			m.Status = "Live quota refreshed"
		} else {
			m.Status = "Ready"
		}
		return m, m.scheduleAutoRefresh()
	case autoRefreshMsg:
		if msg.sequence != m.RefreshSequence || m.Options.AutoRefresh <= 0 || m.Busy || m.Searching || m.EditingRefresh || m.EditingThreshold || m.confirming() {
			return m, nil
		}
		m.Busy = true
		m.Status = "Auto refreshing live quota..."
		return m, m.loadDataCommand(true)
	case operationMsg:
		m.Busy = false
		m.clearConfirmation()
		if msg.err != nil {
			m.Status = "Command failed: " + msg.err.Error()
			return m, m.scheduleAutoRefresh()
		}
		if msg.details != "" {
			m.Details = msg.details
		}
		switch msg.action {
		case actionDoctor:
			m.Status = "Doctor completed"
			return m, m.scheduleAutoRefresh()
		case actionRefresh:
			m.Busy = true
			m.Status = "Refreshing live quota..."
			return m, m.loadDataCommand(true)
		case actionQuit:
			return m, tea.Quit
		default:
			if msg.profile != "" {
				m.Status = successStatus(msg.action, msg.profile)
			} else {
				m.Status = "Command completed"
			}
			if m.Options.ExitAfterSwitch && msg.action == actionSwitchLaunch {
				return m, tea.Quit
			}
			m.Busy = true
			return m, m.loadDataCommand(false)
		}
	case tea.KeyMsg:
		if m.confirming() {
			return m.updateConfirmation(msg)
		}
		if m.EditingRefresh {
			return m.updateRefreshInput(msg)
		}
		if m.EditingThreshold {
			return m.updateThresholdInput(msg)
		}
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

func successStatus(selected action, profile string) string {
	switch selected {
	case actionHotReload:
		return fmt.Sprintf("Hot-switched backend to %s", profile)
	case actionSwitchLaunch:
		return fmt.Sprintf("Restarted Antigravity with %s", profile)
	case actionSwitchOnly:
		return fmt.Sprintf("Activated %s without launching", profile)
	case actionAutoSwitch:
		return fmt.Sprintf("Auto hot-switched to %s", profile)
	case actionPrevious:
		return fmt.Sprintf("Returned to %s", profile)
	default:
		return fmt.Sprintf("Completed %s for %s", selected, profile)
	}
}

func (m Model) beginConfirmation(selected action, profile string) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(profile) == "" {
		m.Status = "Select an account first"
		return m, nil
	}
	m.ConfirmAction = selected
	m.ConfirmProfile = profile
	m.Focus = focusConfirm
	m.RefreshSequence++
	switch selected {
	case actionHotReload:
		m.ConfirmTitle = "Confirm hot switch"
		m.ConfirmBody = fmt.Sprintf("Switch to %s and restart only the language server?\n\nContinue only after the current response and all tool calls have finished. The Antigravity window, files, chat and terminals should stay open.", profile)
	case actionAutoSwitch:
		m.ConfirmTitle = "Confirm auto hot-switch"
		m.ConfirmBody = fmt.Sprintf("The safest recent live-quota candidate is %s.\n\nContinue only when Antigravity is idle. Stale, old, warned and unknown quota are excluded.", profile)
	default:
		m.ConfirmTitle = "Confirm action"
		m.ConfirmBody = fmt.Sprintf("Run %s for %s?", selected, profile)
	}
	m.Status = "Confirmation required"
	m.Details = m.ConfirmBody + "\n\nEnter / Y to continue. Esc / N to cancel."
	return m, nil
}

func (m Model) updateConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "y", "Y":
		action := m.ConfirmAction
		m.Busy = true
		m.Status = "Applying " + strings.ToLower(m.ConfirmTitle) + "..."
		return m, m.operationCommand(action)
	case "esc", "n", "N":
		m.clearConfirmation()
		m.Status = "Action cancelled"
		return m, m.scheduleAutoRefresh()
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) clearConfirmation() {
	m.ConfirmAction = ""
	m.ConfirmProfile = ""
	m.ConfirmTitle = ""
	m.ConfirmBody = ""
	m.Details = ""
	if m.Focus == focusConfirm {
		m.Focus = focusCommands
	}
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Searching = false
		m.Focus = focusAccounts
		return m, m.scheduleAutoRefresh()
	case "enter":
		m.Searching = false
		m.Focus = focusAccounts
		m.SelectedAccount = 0
		m.Status = fmt.Sprintf("Filter: %q", m.Search)
		return m, m.scheduleAutoRefresh()
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

func (m Model) updateRefreshInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.EditingRefresh = false
		m.Focus = focusCommands
		m.Status = "Auto refresh unchanged"
		return m, m.scheduleAutoRefresh()
	case "enter":
		seconds, err := strconv.Atoi(strings.TrimSpace(m.RefreshInput))
		if err != nil || seconds < 0 {
			m.Status = "Enter a whole number of seconds, or 0 to disable"
			return m, nil
		}
		m.Options.AutoRefresh = time.Duration(seconds) * time.Second
		m.RefreshSequence++
		m.EditingRefresh = false
		m.Focus = focusCommands
		if seconds == 0 {
			m.Status = "Auto refresh disabled"
			m.Details = "Auto refresh is off. Press r to refresh manually."
			return m, nil
		}
		m.Status = fmt.Sprintf("Auto refresh set to every %d seconds", seconds)
		m.Details = fmt.Sprintf("Quota will refresh automatically every %d seconds. Select Refresh timer again or press R to change it.", seconds)
		return m, m.scheduleAutoRefresh()
	case "backspace":
		if m.RefreshInput != "" {
			_, size := utf8.DecodeLastRuneInString(m.RefreshInput)
			m.RefreshInput = m.RefreshInput[:len(m.RefreshInput)-size]
		}
		return m, nil
	case "ctrl+u":
		m.RefreshInput = ""
		return m, nil
	}
	text := msg.String()
	if len(text) == 1 && text >= "0" && text <= "9" {
		m.RefreshInput += text
	}
	return m, nil
}

func (m Model) updateThresholdInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.EditingThreshold = false
		m.Focus = focusCommands
		m.Status = "Auto-switch threshold unchanged"
		return m, m.scheduleAutoRefresh()
	case "enter":
		threshold, err := strconv.Atoi(strings.TrimSpace(m.ThresholdInput))
		if err != nil || threshold < 0 || threshold > 100 {
			m.Status = "Enter a whole percentage from 0 to 100"
			return m, nil
		}
		m.Options.AutoThreshold = threshold
		m.EditingThreshold = false
		m.Focus = focusCommands
		m.Status = fmt.Sprintf("Auto-switch threshold set to %d%%", threshold)
		m.Details = "Press a to preview and confirm the recommendation. Auto hot-switch only uses recent live quota."
		m.Busy = true
		return m, m.loadDataCommand(false)
	case "backspace":
		if m.ThresholdInput != "" {
			_, size := utf8.DecodeLastRuneInString(m.ThresholdInput)
			m.ThresholdInput = m.ThresholdInput[:len(m.ThresholdInput)-size]
		}
		return m, nil
	case "ctrl+u":
		m.ThresholdInput = ""
		return m, nil
	}
	text := msg.String()
	if len(text) == 1 && text >= "0" && text <= "9" {
		m.ThresholdInput += text
	}
	return m, nil
}

func (m Model) beginRefreshInput() (tea.Model, tea.Cmd) {
	m.EditingRefresh = true
	m.Focus = focusRefreshInput
	m.RefreshSequence++
	if m.Options.AutoRefresh > 0 {
		m.RefreshInput = strconv.Itoa(int(m.Options.AutoRefresh / time.Second))
	} else {
		m.RefreshInput = "0"
	}
	m.Status = "Set auto-refresh seconds; use 0 to disable"
	return m, nil
}

func (m Model) beginThresholdInput() (tea.Model, tea.Cmd) {
	m.EditingThreshold = true
	m.Focus = focusThresholdInput
	m.RefreshSequence++
	m.ThresholdInput = strconv.Itoa(m.Options.AutoThreshold)
	m.Status = "Set auto-switch threshold from 0 to 100 percent"
	return m, nil
}

func (m Model) autoSwitchRequested() (tea.Model, tea.Cmd) {
	if !m.Decision.Switch || strings.TrimSpace(m.Decision.Selected.Profile) == "" {
		m.Status = "No automatic switch is needed"
		m.Details = m.Decision.Reason
		return m, nil
	}
	return m.beginConfirmation(actionAutoSwitch, m.Decision.Selected.Profile)
}

func (m Model) runSelectedAction(selected action) (tea.Model, tea.Cmd) {
	if selected == actionQuit {
		return m, tea.Quit
	}
	if selected == actionRefresh {
		m.Busy = true
		m.Status = "Refreshing live quota..."
		return m, m.loadDataCommand(true)
	}
	if selected == actionAutoRefresh {
		return m.beginRefreshInput()
	}
	if selected == actionAutoThreshold {
		return m.beginThresholdInput()
	}
	if selected == actionAutoSwitch {
		return m.autoSwitchRequested()
	}
	if selected == actionHotReload {
		profile, ok := m.selectedProfile()
		if !ok {
			m.Status = "Select an account first"
			return m, nil
		}
		return m.beginConfirmation(actionHotReload, profile)
	}
	m.Busy = true
	m.Status = "Running " + strings.ToLower(dashboardCommands[m.SelectedCommand].Label) + "..."
	return m, m.operationCommand(selected)
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
		m.RefreshSequence++
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
			return m.runSelectedAction(dashboardCommands[m.SelectedCommand].Action)
		}
		m.Focus = focusCommands
		m.SelectedCommand = 0
		m.Status = "Account selected; choose an action and press Enter"
	case "s":
		profile, ok := m.selectedProfile()
		if !ok {
			m.Status = "Select an account first"
			return m, nil
		}
		return m.beginConfirmation(actionHotReload, profile)
	case "r":
		m.Busy = true
		m.Status = "Refreshing live quota..."
		return m, m.loadDataCommand(true)
	case "R":
		return m.beginRefreshInput()
	case "A":
		return m.beginThresholdInput()
	case "a":
		return m.autoSwitchRequested()
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
