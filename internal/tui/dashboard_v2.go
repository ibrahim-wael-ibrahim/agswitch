package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ibrahim-wael/agswitch/internal/quota"
)

// dashboardModel is the runtime presentation layer. It deliberately embeds the
// existing Model so account/switching behavior stays in one place while the UI
// can evolve independently.
type dashboardModel struct {
	Model
	ShowModels bool
}

func newDashboardModel(model Model) dashboardModel {
	return dashboardModel{Model: model}
}

func (m dashboardModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if live, ok := message.(liveQuotaMsg); ok {
		m.Busy = false
		if live.data.err != nil {
			m.Status = live.statusText()
			return m, m.scheduleAutoRefresh()
		}
		firstLoad := !m.Initialized
		m.Accounts = live.data.accounts
		m.Results = live.data.results
		m.Decision = live.data.decision
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
		m.Status = live.statusText()
		return m, m.scheduleAutoRefresh()
	}

	if key, ok := message.(tea.KeyMsg); ok && !m.Busy && !m.Searching && !m.EditingRefresh && !m.EditingThreshold && !m.confirming() {
		switch key.String() {
		case "m":
			m.ShowModels = !m.ShowModels
			if m.ShowModels {
				m.Status = "Model details shown"
			} else {
				m.Status = "Model details hidden"
			}
			return m, nil
		case "r":
			m.Busy = true
			m.Status = "Refreshing all account quotas..."
			return m, m.loadLiveQuotaCommand()
		}
	}

	if tick, ok := message.(autoRefreshMsg); ok {
		if tick.sequence == m.RefreshSequence && m.Options.AutoRefresh > 0 && !m.Busy && !m.Searching && !m.EditingRefresh && !m.EditingThreshold && !m.confirming() {
			m.Busy = true
			m.Status = "Refreshing all account quotas..."
			return m, m.loadLiveQuotaCommand()
		}
	}

	updated, cmd := m.Model.Update(message)
	base, ok := updated.(Model)
	if !ok {
		return updated, cmd
	}
	m.Model = base
	return m, cmd
}

func (m dashboardModel) View() tea.View {
	width, height := m.Width, m.Height
	if width <= 0 {
		width = 110
	}
	if height <= 0 {
		height = 34
	}
	width = max(42, width-2)
	theme := currentTheme()

	if m.confirming() {
		parts := []string{
			renderFriendlyHeader(width, m.Model, theme),
			renderConfirmationPanelWithTheme(m.Model, width, min(max(9, height-8), 13), theme),
			renderFriendlyHelp(width, m, theme),
		}
		view := tea.NewView(strings.Join(parts, "\n"))
		view.AltScreen = true
		return view
	}

	header := renderFriendlyHeader(width, m.Model, theme)
	footer := renderFriendlyHelp(width, m, theme)
	available := max(12, height-7)
	accountsMin := min(max(len(m.filteredAccounts())+4, 9), 16)

	var body string
	if width >= 116 {
		accountsWidth := max(66, width*60/100)
		detailWidth := max(46, width-accountsWidth-1)
		panelHeight := min(max(accountsMin, 14), available)
		body = lipgloss.JoinHorizontal(
			lipgloss.Top,
			renderFriendlyAccounts(m.Model, accountsWidth, panelHeight, theme),
			" ",
			renderAccountHealth(m.Model, detailWidth, panelHeight, theme),
		)
	} else {
		accountHeight := min(accountsMin, max(9, available/2))
		detailHeight := max(9, available-accountHeight-1)
		body = renderFriendlyAccounts(m.Model, width, accountHeight, theme) + "\n" + renderAccountHealth(m.Model, width, detailHeight, theme)
	}

	parts := []string{header, body}
	if m.ShowModels && height >= 25 {
		modelHeight := max(7, min(12, height-lipgloss.Height(header)-lipgloss.Height(body)-3))
		parts = append(parts, renderCompactModels(m.Model, width, modelHeight, theme))
	}
	parts = append(parts, footer)

	view := tea.NewView(strings.Join(parts, "\n"))
	view.AltScreen = true
	return view
}

func renderFriendlyHeader(width int, m Model, theme tuiTheme) string {
	active := m.activeProfile()
	if active == "" {
		active = "unknown"
	}
	refresh := "manual"
	if m.Options.AutoRefresh > 0 {
		refresh = compactDuration(m.Options.AutoRefresh)
	}

	title := theme.style(theme.Accent).Bold(true).Render("AGSwitch")
	version := theme.style(theme.Muted).Render(" " + m.Options.Version)
	statusColor := theme.Success
	statusIcon := "●"
	if m.Busy {
		statusColor = theme.Warning
		statusIcon = "◌"
	}
	low := strings.ToLower(m.Status)
	if strings.Contains(low, "failed") || strings.Contains(low, "error") {
		statusColor = theme.Danger
		statusIcon = "!"
	}
	status := theme.style(statusColor).Bold(true).Render(statusIcon + " " + sanitizeTerminalText(m.Status))

	line1 := title + version + "  " + theme.style(theme.Muted).Render("Antigravity account dashboard")
	line2 := chip(theme, "ACTIVE", active, theme.Accent) + "  " + chip(theme, "LIVE REFRESH", refresh, theme.Info) + "  " + chip(theme, "AUTO", fmt.Sprintf("%d%%", m.Options.AutoThreshold), theme.Warning)
	return trimWidth(line1, width) + "\n" + trimWidth(line2, width) + "\n" + trimWidth(status, width)
}

func renderFriendlyAccounts(m Model, width, height int, theme tuiTheme) string {
	items := m.filteredAccounts()
	now := time.Now().UTC()
	lines := make([]string, 0, len(items)+2)
	searchValue := m.Search
	if m.Searching {
		searchValue += "_"
	}
	lines = append(lines, theme.style(theme.Info).Bold(true).Render("/ Search")+theme.style(theme.Muted).Render(": "+searchValue))

	if len(items) == 0 {
		lines = append(lines, theme.style(theme.Muted).Render("No matching accounts"))
	}
	for index, item := range items {
		result := m.resultFor(item.ID)
		live, state := dashboardQuotaStatus(result, now)
		windows := quota.SummarizeWindows(result.Snapshot, now)

		active := "○"
		if item.Active {
			active = "●"
		}
		status := renderDashboardBadge(state, theme)
		five := ""
		if live && windows.FiveHour.Available {
			five = renderWindowInline("5h", windows.FiveHour, now, theme)
		} else if live {
			if remaining, ok := quota.MinimumKnownRemaining(result.Snapshot); ok {
				five = theme.style(quotaThemeColor(theme, remaining)).Bold(true).Render(fmt.Sprintf("%d%%", remaining))
			}
		}

		right := status
		if five != "" {
			right += "  " + five
		}
		leftWidth := max(10, width-visibleWidth(right)-9)
		left := active + " " + padRight(trimWidth(item.ID, leftWidth), leftWidth)
		line := theme.style(theme.Text).Render(active+" ") + theme.style(theme.Text).Bold(item.Active).Render(padRight(trimWidth(item.ID, leftWidth), leftWidth)) + "  " + right
		if m.Focus == focusAccounts && index == m.SelectedAccount {
			line = selectedLine(theme, left+"  "+stripANSI(right), max(1, width-6))
		}
		lines = append(lines, line)
	}

	return renderBoxWithTheme("Accounts · five-hour availability", strings.Join(lines, "\n"), width, height, m.Focus == focusAccounts || m.Focus == focusSearch, theme)
}

func renderDashboardBadge(state string, theme tuiTheme) string {
	var color = theme.Muted
	switch state {
	case "LIVE":
		color = theme.Success
	case "AUTH":
		color = theme.Danger
	case "RETRY":
		color = theme.Info
	case "STALE", "OLD":
		color = theme.Warning
	}
	return theme.style(color).Bold(true).Render("[" + state + "]")
}

func renderAccountHealth(m Model, width, height int, theme tuiTheme) string {
	profile, ok := m.selectedProfile()
	if !ok {
		return renderBoxWithTheme("Selected account", theme.style(theme.Muted).Render("Select an account."), width, height, false, theme)
	}
	result := m.resultFor(profile)
	now := time.Now().UTC()
	live, state := dashboardQuotaStatus(result, now)
	windows := quota.SummarizeWindows(result.Snapshot, now)

	lines := []string{theme.style(theme.Text).Bold(true).Render(profile) + "  " + renderDashboardBadge(state, theme)}
	for _, item := range m.Accounts {
		if item.ID == profile && strings.TrimSpace(item.Email) != "" {
			lines = append(lines, theme.style(theme.Muted).Render(item.Email))
			break
		}
	}
	lines = append(lines, "")

	if live && windows.FiveHour.Available {
		lines = append(lines, renderWindowBlock("5 HOUR", windows.FiveHour, now, width-6, theme))
	} else if live {
		lines = append(lines, theme.style(theme.Warning).Bold(true).Render("5 HOUR  provider did not expose a short reset window"))
	} else if state == "AUTH" {
		lines = append(lines, theme.style(theme.Danger).Bold(true).Render("5 HOUR  needs refreshed authentication for this saved profile"))
	} else {
		lines = append(lines, theme.style(theme.Warning).Bold(true).Render("5 HOUR  waiting for a live provider response"))
	}

	lines = append(lines, "")
	if live && windows.Weekly.Available {
		lines = append(lines, renderWindowBlock("WEEKLY", windows.Weekly, now, width-6, theme))
	} else {
		lines = append(lines, theme.style(theme.Muted).Render("WEEKLY  provider did not expose a separate long reset window"))
	}

	if !live {
		if warning := strings.TrimSpace(result.Snapshot.Metadata["warning"]); warning != "" {
			lines = append(lines, "", theme.style(theme.Warning).Render(trimWidth(warning, width-6)))
		} else if result.Err != nil {
			lines = append(lines, "", theme.style(theme.Warning).Render(trimWidth(result.Err.Error(), width-6)))
		}
	}

	lines = append(lines, "", theme.style(theme.Muted).Render("Enter/s hot switch · r refresh all · a auto · m model details"))
	return renderBoxWithTheme("Selected account", strings.Join(lines, "\n"), width, height, false, theme)
}

func renderWindowInline(label string, window quota.WindowSummary, now time.Time, theme tuiTheme) string {
	c := quotaThemeColor(theme, window.Remaining)
	remaining := theme.style(c).Bold(true).Render(fmt.Sprintf("%s %d%%", label, window.Remaining))
	if window.Exhausted {
		remaining = theme.style(theme.Danger).Bold(true).Render(label + " EXHAUSTED")
	}
	if window.ResetAt.IsZero() {
		return remaining
	}
	return remaining + theme.style(theme.Muted).Render(" · "+compactDuration(maxDuration(0, window.ResetAt.Sub(now))))
}

func renderWindowBlock(label string, window quota.WindowSummary, now time.Time, width int, theme tuiTheme) string {
	c := quotaThemeColor(theme, window.Remaining)
	if window.Exhausted {
		c = theme.Danger
	}
	barWidth := min(22, max(8, width-22))
	bar := theme.style(c).Render(compactBar(window.Remaining, barWidth))
	value := theme.style(c).Bold(true).Render(fmt.Sprintf("%3d%%", window.Remaining))
	if window.Exhausted {
		value = theme.style(theme.Danger).Bold(true).Render("0% EXHAUSTED")
	}
	reset := "reset unknown"
	if !window.ResetAt.IsZero() {
		reset = "resets in " + compactDuration(maxDuration(0, window.ResetAt.Sub(now)))
	}
	return theme.style(theme.Muted).Bold(true).Render(label+"  ") + bar + " " + value + "\n" + theme.style(theme.Muted).Render("       "+reset)
}

func renderCompactModels(m Model, width, height int, theme tuiTheme) string {
	profile, ok := m.selectedProfile()
	if !ok {
		return ""
	}
	result := m.resultFor(profile)
	models := quota.SortedModels(result.Snapshot)
	if len(models) == 0 {
		return renderBoxWithTheme("Model details", theme.style(theme.Muted).Render("No model quota returned."), width, height, false, theme)
	}

	maxRows := max(1, height-3)
	if len(models) > maxRows {
		models = models[:maxRows]
	}
	lines := make([]string, 0, len(models))
	for _, model := range models {
		remaining := "unknown"
		if model.Remaining >= 0 {
			remaining = fmt.Sprintf("%d%%", model.Remaining)
		}
		reset := ""
		if d := quota.ResetIn(model.ResetAt, time.Now()); d > 0 {
			reset = " · " + compactDuration(d)
		}
		nameWidth := max(10, width-28)
		lines = append(lines, fmt.Sprintf("%-*s %8s%s", nameWidth, trimWidth(model.Name, nameWidth), remaining, reset))
	}
	return renderBoxWithTheme("Model details · m to hide", strings.Join(lines, "\n"), width, height, false, theme)
}

func renderFriendlyHelp(width int, m dashboardModel, theme tuiTheme) string {
	text := "↑/↓ account  Enter actions  s hot-switch  r refresh  a auto  / search  m models  Tab panels  q quit"
	if m.confirming() {
		text = "Enter / Y confirm · Esc / N cancel · q quit"
	} else if m.EditingRefresh {
		text = "Type seconds · Enter save · Esc cancel"
	} else if m.EditingThreshold {
		text = "Type threshold 0–100 · Enter save · Esc cancel"
	} else if m.Searching {
		text = "Type to search · Enter apply · Ctrl-U clear · Esc cancel"
	}
	return theme.style(theme.Muted).Render(trimWidth(text, width))
}
