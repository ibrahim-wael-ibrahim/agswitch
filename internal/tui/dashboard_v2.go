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
	accountsMin := min(max(len(m.filteredAccounts())+5, 10), 17)

	var body string
	if width >= 116 {
		// Accounts need enough width for a name + state + five-hour summary, but
		// the selected-account panel benefits more from extra room because auth
		// and provider explanations are prose rather than columns.
		accountsWidth := max(58, width*55/100)
		detailWidth := max(50, width-accountsWidth-1)
		panelHeight := min(max(accountsMin, 14), available)
		body = lipgloss.JoinHorizontal(
			lipgloss.Top,
			renderFriendlyAccounts(m.Model, accountsWidth, panelHeight, theme),
			" ",
			renderAccountHealth(m.Model, detailWidth, panelHeight, theme),
		)
	} else {
		accountHeight := min(accountsMin, max(10, available/2))
		detailHeight := max(10, available-accountHeight-1)
		body = renderFriendlyAccounts(m.Model, width, accountHeight, theme) + "\n" + renderAccountHealth(m.Model, width, detailHeight, theme)
	}

	parts := []string{header, body}
	if m.ShowModels && height >= 25 {
		modelHeight := max(7, min(14, height-lipgloss.Height(header)-lipgloss.Height(body)-3))
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
	lines := make([]string, 0, len(items)+3)
	searchValue := m.Search
	if m.Searching {
		searchValue += "_"
	}
	lines = append(lines, theme.style(theme.Info).Bold(true).Render("/ Search")+theme.style(theme.Muted).Render(": "+searchValue))

	contentWidth := max(12, width-6)
	if contentWidth >= 58 {
		leftHeader := "ACCOUNT"
		rightHeader := "STATE / FIVE-HOUR"
		gap := max(2, contentWidth-visibleWidth(leftHeader)-visibleWidth(rightHeader))
		lines = append(lines, theme.style(theme.Muted).Render(leftHeader+strings.Repeat(" ", gap)+rightHeader))
	}

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
		// The box content area is width-6. Previous revisions budgeted against
		// the outer width, causing the final badge to wrap onto a second line.
		leftWidth := max(8, contentWidth-visibleWidth(right)-4)
		name := trimWidth(item.ID, leftWidth)
		leftPlain := active + " " + padRight(name, leftWidth)
		plain := leftPlain + "  " + stripANSI(right)
		line := theme.style(theme.Text).Render(active+" ") + theme.style(theme.Text).Bold(item.Active).Render(padRight(name, leftWidth)) + "  " + right
		if m.Focus == focusAccounts && index == m.SelectedAccount {
			line = selectedLine(theme, plain, contentWidth)
		} else {
			line = trimWidth(line, contentWidth)
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
	contentWidth := max(12, width-6)

	lines := []string{theme.style(theme.Text).Bold(true).Render(profile) + "  " + renderDashboardBadge(state, theme)}
	for _, item := range m.Accounts {
		if item.ID == profile && strings.TrimSpace(item.Email) != "" {
			lines = append(lines, theme.style(theme.Muted).Render(trimWidth(item.Email, contentWidth)))
			break
		}
	}
	lines = append(lines, "")

	if live && windows.FiveHour.Available {
		lines = append(lines, renderWindowBlock("5 HOUR", windows.FiveHour, now, contentWidth, theme))
	} else if live {
		lines = appendStyledWrapped(lines, "5 HOUR  provider did not expose a short reset window", contentWidth, theme.style(theme.Warning).Bold(true))
	} else if state == "AUTH" {
		lines = appendStyledWrapped(lines, "5 HOUR  needs refreshed authentication for this saved profile", contentWidth, theme.style(theme.Danger).Bold(true))
	} else {
		lines = appendStyledWrapped(lines, "5 HOUR  waiting for a live provider response", contentWidth, theme.style(theme.Warning).Bold(true))
	}

	lines = append(lines, "")
	if live && windows.Weekly.Available {
		lines = append(lines, renderWindowBlock("WEEKLY", windows.Weekly, now, contentWidth, theme))
	} else {
		lines = appendStyledWrapped(lines, "WEEKLY  provider did not expose a separate long reset window", contentWidth, theme.style(theme.Muted))
	}

	if !live {
		warning := strings.TrimSpace(result.Snapshot.Metadata["warning"])
		if warning == "" && result.Err != nil {
			warning = result.Err.Error()
		}
		if warning != "" {
			lines = append(lines, "")
			prefix := "DETAIL  "
			if state == "AUTH" {
				prefix = "AUTH    "
			}
			lines = appendStyledWrapped(lines, prefix+warning, contentWidth, theme.style(theme.Warning))
		}
	}

	lines = append(lines, "", theme.style(theme.Muted).Render(trimWidth("Enter/s switch · r refresh · a auto · m models", contentWidth)))
	return renderBoxWithTheme("Selected account", strings.Join(lines, "\n"), width, height, false, theme)
}

func appendStyledWrapped(lines []string, text string, width int, style lipgloss.Style) []string {
	for _, line := range wrapPlainText(text, width) {
		lines = append(lines, style.Render(line))
	}
	return lines
}

func wrapPlainText(value string, width int) []string {
	value = strings.TrimSpace(stripANSI(sanitizeTerminalText(value)))
	if value == "" {
		return nil
	}
	width = max(1, width)
	words := strings.Fields(value)
	lines := make([]string, 0, 2)
	current := ""
	for _, word := range words {
		if visibleWidth(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			for visibleWidth(word) > width {
				piece := trimWidth(word, width)
				piece = strings.TrimSuffix(piece, "…")
				if piece == "" {
					break
				}
				lines = append(lines, piece)
				word = strings.TrimPrefix(word, piece)
			}
			if word != "" {
				current = word
			}
			continue
		}
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if visibleWidth(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
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

	contentWidth := max(16, width-6)
	rowsAvailable := max(1, height-3)
	columns := 1
	if contentWidth >= 104 {
		columns = 2
	}
	capacity := rowsAvailable * columns
	shown := len(models)
	if shown > capacity {
		shown = capacity
	}
	models = models[:shown]

	cellWidth := contentWidth
	if columns == 2 {
		cellWidth = (contentWidth - 2) / 2
	}
	entries := make([]string, 0, len(models))
	for _, model := range models {
		remaining := "unknown"
		if model.Remaining >= 0 {
			remaining = fmt.Sprintf("%d%%", model.Remaining)
		}
		reset := ""
		if d := quota.ResetIn(model.ResetAt, time.Now()); d > 0 {
			reset = " · " + compactDuration(d)
		}
		right := remaining + reset
		nameWidth := max(8, cellWidth-visibleWidth(right)-1)
		entry := padRight(trimWidth(model.Name, nameWidth), nameWidth) + " " + right
		entries = append(entries, trimWidth(entry, cellWidth))
	}

	var body string
	if columns == 1 {
		body = strings.Join(entries, "\n")
	} else {
		rows := (len(entries) + 1) / 2
		lines := make([]string, 0, rows)
		for row := 0; row < rows; row++ {
			left := entries[row]
			right := ""
			if row+rows < len(entries) {
				right = entries[row+rows]
			}
			lines = append(lines, padRight(left, cellWidth)+"  "+right)
		}
		body = strings.Join(lines, "\n")
	}

	title := "Model details · m to hide"
	if shown < len(quota.SortedModels(result.Snapshot)) {
		title = fmt.Sprintf("Model details · %d/%d shown · m to hide", shown, len(quota.SortedModels(result.Snapshot)))
	}
	return renderBoxWithTheme(title, body, width, height, false, theme)
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
