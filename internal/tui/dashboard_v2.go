package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ibrahim-wael/agswitch/internal/quota"
)

const dashboardWidthSafety = 3

// dashboardModel is the runtime presentation layer. It deliberately embeds the
// existing Model so account/switching behavior stays in one place while the UI
// can evolve independently.
type dashboardModel struct {
	Model
	ShowModels    bool
	ShowAllModels bool
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
			m.ShowAllModels = false
			if m.ShowModels {
				m.Status = "Model summary shown"
			} else {
				m.Status = "Model details hidden"
			}
			return m, nil
		case "M":
			m.ShowModels = true
			m.ShowAllModels = !m.ShowAllModels
			if m.ShowAllModels {
				m.Status = "All model rows shown"
			} else {
				m.Status = "Model summary shown"
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
		confirmWidth := min(width, 96)
		confirmation := renderConfirmationPanelWithTheme(m.Model, confirmWidth, 10, theme)
		if confirmWidth < width {
			confirmation = lipgloss.PlaceHorizontal(width, lipgloss.Center, confirmation)
		}
		parts := []string{
			renderFriendlyHeader(width, m.Model, theme),
			confirmation,
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
		parts = append(parts, renderModelsPanel(m, width, modelHeight, theme))
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

	rowWidth := dashboardContentWidth(width)
	if rowWidth >= 54 {
		leftHeader := "ACCOUNT"
		rightHeader := "STATUS / 5H RESET"
		gap := max(2, rowWidth-visibleWidth(leftHeader)-visibleWidth(rightHeader))
		lines = append(lines, theme.style(theme.Muted).Render(leftHeader+strings.Repeat(" ", gap)+rightHeader))
	}

	if len(items) == 0 {
		lines = append(lines, theme.style(theme.Muted).Render("No matching accounts"))
	}
	for index, item := range items {
		result := m.resultFor(item.ID)
		line, plain := renderFriendlyAccountRow(item.ID, item.Active, result, rowWidth, now, theme)
		if m.Decision.Switch && m.Decision.Selected.Profile == item.ID {
			best := theme.style(theme.Warning).Bold(true).Render("★")
			if visibleWidth(line)+2 <= rowWidth {
				line = best + " " + line
				plain = "★ " + plain
			}
		}
		if m.Focus == focusAccounts && index == m.SelectedAccount {
			line = selectedLine(theme, trimWidth(plain, rowWidth), rowWidth)
		}
		lines = append(lines, line)
	}

	return renderBoxWithTheme("Accounts · five-hour availability", strings.Join(lines, "\n"), width, height, m.Focus == focusAccounts || m.Focus == focusSearch, theme)
}

func renderFriendlyAccountRow(profile string, active bool, result quota.Result, rowWidth int, now time.Time, theme tuiTheme) (string, string) {
	live, state := dashboardQuotaStatus(result, now)
	windows := quota.SummarizeWindows(result.Snapshot, now)
	marker := "○"
	if active {
		marker = "●"
	}

	rightPlain := "[" + state + "]"
	rightStyled := renderDashboardBadge(state, theme)
	if live && windows.FiveHour.Available {
		windowPlain := compactWindowInline("5h", windows.FiveHour, now)
		rightPlain += "  " + windowPlain
		rightStyled += "  " + renderCompactWindowInline("5h", windows.FiveHour, now, theme)
	} else if live {
		if remaining, ok := quota.MinimumKnownRemaining(result.Snapshot); ok {
			value := fmt.Sprintf("%d%%", remaining)
			rightPlain += "  " + value
			rightStyled += "  " + theme.style(quotaThemeColor(theme, remaining)).Bold(true).Render(value)
		}
	}

	rowWidth = max(12, rowWidth-dashboardWidthSafety)
	leftWidth := max(8, rowWidth-visibleWidth(rightPlain)-4)
	name := trimWidth(profile, leftWidth)
	leftPlain := marker + " " + padRight(name, leftWidth)
	plain := trimWidth(leftPlain+"  "+rightPlain, rowWidth)
	styled := theme.style(theme.Text).Render(marker+" ") + theme.style(theme.Text).Bold(active).Render(padRight(name, leftWidth)) + "  " + rightStyled
	return trimWidth(styled, rowWidth), plain
}

func dashboardContentWidth(outerWidth int) int {
	return max(12, outerWidth-9)
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
	contentWidth := dashboardContentWidth(width)

	lines := []string{theme.style(theme.Text).Bold(true).Render(profile) + "  " + renderDashboardBadge(state, theme)}
	for _, item := range m.Accounts {
		if item.ID == profile && strings.TrimSpace(item.Email) != "" {
			lines = append(lines, theme.style(theme.Muted).Render(trimWidth(item.Email, contentWidth)))
			break
		}
	}

	meta := quotaMetaLine(result, now)
	if meta != "" {
		lines = append(lines, theme.style(theme.Muted).Render(trimWidth(meta, contentWidth)))
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
		lines = appendStyledWrapped(lines, "WEEKLY provider did not expose a separate long reset window", contentWidth, theme.style(theme.Muted))
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

	lines = append(lines, "", theme.style(theme.Muted).Render(trimWidth("Enter/s switch · r refresh · a auto · m summary · M all", contentWidth)))
	return renderBoxWithTheme("Selected account", strings.Join(lines, "\n"), width, height, false, theme)
}

func quotaMetaLine(result quota.Result, now time.Time) string {
	parts := make([]string, 0, 3)
	if !result.Snapshot.FetchedAt.IsZero() {
		age := maxDuration(0, now.Sub(result.Snapshot.FetchedAt.UTC()))
		parts = append(parts, "updated "+compactAge(age)+" ago")
	}
	if count := len(result.Snapshot.Models); count > 0 {
		parts = append(parts, fmt.Sprintf("%d models", count))
	}
	if source := strings.TrimSpace(result.Snapshot.Source); source != "" {
		parts = append(parts, source)
	}
	return strings.Join(parts, " · ")
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

func renderCompactWindowInline(label string, window quota.WindowSummary, now time.Time, theme tuiTheme) string {
	c := quotaThemeColor(theme, window.Remaining)
	value := fmt.Sprintf("%s %d%%", label, window.Remaining)
	if window.Exhausted {
		value = label + " EXHAUSTED"
		c = theme.Danger
	}
	styled := theme.style(c).Bold(true).Render(value)
	if window.ResetAt.IsZero() {
		return styled
	}
	return styled + theme.style(theme.Muted).Render(" · "+compactResetDuration(maxDuration(0, window.ResetAt.Sub(now))))
}

func compactWindowInline(label string, window quota.WindowSummary, now time.Time) string {
	value := fmt.Sprintf("%s %d%%", label, window.Remaining)
	if window.Exhausted {
		value = label + " EXHAUSTED"
	}
	if window.ResetAt.IsZero() {
		return value
	}
	return value + " · " + compactResetDuration(maxDuration(0, window.ResetAt.Sub(now)))
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

type modelGroup struct {
	Remaining int
	Reset     string
	Count     int
	Names     []string
}

func renderModelsPanel(m dashboardModel, width, height int, theme tuiTheme) string {
	profile, ok := m.selectedProfile()
	if !ok {
		return ""
	}
	result := m.resultFor(profile)
	live, state := dashboardQuotaStatus(result, time.Now().UTC())
	allModels := quota.SortedModels(result.Snapshot)
	if len(allModels) == 0 {
		return renderBoxWithTheme("Models", theme.style(theme.Muted).Render("No model quota returned."), width, height, false, theme)
	}

	if !live && !m.ShowAllModels {
		age := "unknown age"
		if !result.Snapshot.FetchedAt.IsZero() {
			age = compactAge(maxDuration(0, time.Since(result.Snapshot.FetchedAt))) + " old"
		}
		lines := []string{
			theme.style(theme.Warning).Bold(true).Render("Cached model snapshot · display only"),
			theme.style(theme.Muted).Render(fmt.Sprintf("%s · %d cached model rows · %s", state, len(allModels), age)),
			theme.style(theme.Muted).Render("Live percentages are hidden because this profile is not authenticated."),
			theme.style(theme.Info).Render("Press M to inspect the cached rows explicitly."),
		}
		return renderBoxWithTheme("Models · cached", strings.Join(lines, "\n"), width, min(height, 7), false, theme)
	}

	if !m.ShowAllModels {
		return renderModelSummary(result, allModels, width, height, theme)
	}
	return renderAllModelRows(result, allModels, width, height, live, theme)
}

func renderModelSummary(result quota.Result, models []quota.ModelUsage, width, height int, theme tuiTheme) string {
	now := time.Now()
	groups := make(map[string]*modelGroup)
	for _, model := range models {
		remaining := model.Remaining
		reset := "—"
		if d := quota.ResetIn(model.ResetAt, now); d > 0 {
			reset = compactResetDuration(d)
		}
		key := fmt.Sprintf("%d|%s", remaining, reset)
		group := groups[key]
		if group == nil {
			group = &modelGroup{Remaining: remaining, Reset: reset}
			groups[key] = group
		}
		group.Count++
		if len(group.Names) < 3 {
			group.Names = append(group.Names, model.Name)
		}
	}

	ordered := make([]*modelGroup, 0, len(groups))
	for _, group := range groups {
		ordered = append(ordered, group)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Remaining < 0 && ordered[j].Remaining >= 0 {
			return false
		}
		if ordered[j].Remaining < 0 && ordered[i].Remaining >= 0 {
			return true
		}
		if ordered[i].Remaining != ordered[j].Remaining {
			return ordered[i].Remaining < ordered[j].Remaining
		}
		return ordered[i].Reset < ordered[j].Reset
	})

	contentWidth := dashboardContentWidth(width)
	lines := make([]string, 0, len(ordered)+1)
	for _, group := range ordered {
		value := "—"
		if group.Remaining >= 0 {
			value = fmt.Sprintf("%d%%", group.Remaining)
		}
		preview := strings.Join(group.Names, ", ")
		if group.Count > len(group.Names) {
			preview += fmt.Sprintf(" +%d", group.Count-len(group.Names))
		}
		right := fmt.Sprintf("%s  %s  ×%d", value, group.Reset, group.Count)
		leftWidth := max(10, contentWidth-visibleWidth(right)-3)
		lines = append(lines, padRight(trimWidth(preview, leftWidth), leftWidth)+"   "+right)
	}

	title := fmt.Sprintf("Model summary · %d models · %d groups · M all", len(models), len(ordered))
	return renderBoxWithTheme(title, strings.Join(lines, "\n"), width, min(height, max(6, len(lines)+2)), false, theme)
}

func renderAllModelRows(result quota.Result, allModels []quota.ModelUsage, width, height int, live bool, theme tuiTheme) string {
	contentWidth := dashboardContentWidth(width)
	rowsAvailable := max(1, height-4)
	columns := 1
	if contentWidth >= 104 {
		columns = 2
	}
	capacity := rowsAvailable * columns
	shown := min(len(allModels), capacity)
	models := allModels[:shown]

	cellWidth := contentWidth
	if columns == 2 {
		cellWidth = max(18, (contentWidth-5)/2)
	}
	entries := make([]string, 0, len(models))
	for _, model := range models {
		entries = append(entries, renderModelCell(model, cellWidth, time.Now(), live))
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
			lines = append(lines, padRight(left, cellWidth)+"   "+right)
		}
		body = strings.Join(lines, "\n")
	}

	mode := "LIVE"
	if !live {
		mode = "CACHED / DISPLAY ONLY"
	}
	title := fmt.Sprintf("All models · %s · %d/%d · M summary", mode, shown, len(allModels))
	return renderBoxWithTheme(title, body, width, height, false, theme)
}

func renderModelCell(model quota.ModelUsage, width int, now time.Time, showValues bool) string {
	remaining := "—"
	reset := "—"
	if showValues {
		if model.Remaining >= 0 {
			remaining = fmt.Sprintf("%d%%", model.Remaining)
		}
		if d := quota.ResetIn(model.ResetAt, now); d > 0 {
			reset = compactResetDuration(d)
		}
	} else {
		remaining = "cached"
		if d := quota.ResetIn(model.ResetAt, now); d > 0 {
			reset = compactResetDuration(d)
		}
	}
	right := remaining + "  " + reset
	nameWidth := max(7, width-visibleWidth(right)-2)
	name := trimWidth(model.Name, nameWidth)
	return trimWidth(padRight(name, nameWidth)+"  "+right, width)
}

func compactResetDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	if value >= 24*time.Hour {
		return fmt.Sprintf("%dd%02dh", int(value/(24*time.Hour)), int(value%(24*time.Hour)/time.Hour))
	}
	if value >= time.Hour {
		return fmt.Sprintf("%dh%02dm", int(value/time.Hour), int(value%time.Hour/time.Minute))
	}
	if value >= time.Minute {
		return fmt.Sprintf("%dm", max(1, int(value/time.Minute)))
	}
	return fmt.Sprintf("%ds", max(1, int(value/time.Second)))
}

func compactAge(value time.Duration) string {
	if value < time.Minute {
		return fmt.Sprintf("%ds", max(0, int(value/time.Second)))
	}
	return compactResetDuration(value)
}

func renderFriendlyHelp(width int, m dashboardModel, theme tuiTheme) string {
	text := "↑/↓ account  Enter/s switch  f restart  u sync  o activate  r refresh  a auto  / search  m summary  M all  q quit"
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
