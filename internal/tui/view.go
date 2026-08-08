package tui

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ibrahim-wael/agswitch/internal/brand"
	"github.com/ibrahim-wael/agswitch/internal/quota"
)

type tuiTheme struct {
	Accent        color.Color
	AccentSoft    color.Color
	Success       color.Color
	Warning       color.Color
	Danger        color.Color
	Info          color.Color
	Text          color.Color
	Muted         color.Color
	Border        color.Color
	Surface       color.Color
	SelectedBG    color.Color
	SelectedText  color.Color
	ColorsEnabled bool
}

func currentTheme() tuiTheme {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AGSWITCH_THEME")))
	dark := mode != "light"
	pick := lipgloss.LightDark(dark)
	colors := os.Getenv("NO_COLOR") == "" && strings.ToLower(os.Getenv("TERM")) != "dumb"
	return tuiTheme{
		Accent:        pick(lipgloss.Color("#5B21B6"), lipgloss.Color("#A78BFA")),
		AccentSoft:    pick(lipgloss.Color("#EDE9FE"), lipgloss.Color("#2E1065")),
		Success:       pick(lipgloss.Color("#047857"), lipgloss.Color("#34D399")),
		Warning:       pick(lipgloss.Color("#B45309"), lipgloss.Color("#FBBF24")),
		Danger:        pick(lipgloss.Color("#B91C1C"), lipgloss.Color("#F87171")),
		Info:          pick(lipgloss.Color("#0369A1"), lipgloss.Color("#38BDF8")),
		Text:          pick(lipgloss.Color("#111827"), lipgloss.Color("#F8FAFC")),
		Muted:         pick(lipgloss.Color("#6B7280"), lipgloss.Color("#94A3B8")),
		Border:        pick(lipgloss.Color("#D1D5DB"), lipgloss.Color("#334155")),
		Surface:       pick(lipgloss.Color("#F8FAFC"), lipgloss.Color("#0F172A")),
		SelectedBG:    pick(lipgloss.Color("#DDD6FE"), lipgloss.Color("#4C1D95")),
		SelectedText:  pick(lipgloss.Color("#2E1065"), lipgloss.Color("#FFFFFF")),
		ColorsEnabled: colors,
	}
}

func (t tuiTheme) style(c color.Color) lipgloss.Style {
	s := lipgloss.NewStyle()
	if t.ColorsEnabled {
		s = s.Foreground(c)
	}
	return s
}

func (m Model) View() tea.View {
	width, height := m.Width, m.Height
	if width <= 0 {
		width = 110
	}
	if height <= 0 {
		height = 34
	}
	width = max(42, width-2)
	theme := currentTheme()

	parts := []string{renderHeaderWithTheme(width, m, theme)}
	if m.confirming() {
		parts = append(parts, renderConfirmationPanelWithTheme(m, width, min(max(9, height-8), 13), theme))
		parts = append(parts, renderHelpWithTheme(m, width, theme))
		view := tea.NewView(strings.Join(parts, "\n"))
		view.AltScreen = true
		return view
	}

	commandHeight := min(max(len(dashboardCommands)+2, 12), 16)
	accountHeight := min(max(len(m.filteredAccounts())*2+3, 12), 16)
	topHeight := max(commandHeight, accountHeight)
	if width >= 94 {
		leftWidth := width/2 - 1
		rightWidth := width - leftWidth - 1
		parts = append(parts, lipgloss.JoinHorizontal(
			lipgloss.Top,
			renderCommandPanelWithTheme(m, leftWidth, topHeight, theme),
			" ",
			renderAccountPanelWithTheme(m, rightWidth, topHeight, theme),
		))
	} else {
		parts = append(parts, renderAccountPanelWithTheme(m, width, accountHeight, theme))
		parts = append(parts, renderCommandPanelWithTheme(m, width, commandHeight, theme))
		topHeight = commandHeight + accountHeight + 1
	}

	reserved := topHeight + 9
	if strings.TrimSpace(m.Details) != "" {
		reserved += 6
	}
	parts = append(parts, renderQuotaPanelWithTheme(m, width, min(max(7, height-reserved), 15), theme))
	if strings.TrimSpace(m.Details) != "" {
		parts = append(parts, renderBoxWithTheme("Details", truncateLines(m.Details, 4, width-6), width, 6, false, theme))
	}
	parts = append(parts, renderHelpWithTheme(m, width, theme))

	view := tea.NewView(strings.Join(parts, "\n"))
	view.AltScreen = true
	return view
}

func renderHeader(width int, m Model) string {
	return renderHeaderWithTheme(width, m, currentTheme())
}

func renderHeaderWithTheme(width int, m Model, theme tuiTheme) string {
	active := m.activeProfile()
	if active == "" {
		active = "unknown"
	}
	refresh := "manual"
	if m.Options.AutoRefresh > 0 {
		refresh = "every " + compactDuration(m.Options.AutoRefresh)
	}

	title := theme.style(theme.Accent).Bold(true).Render("AGSwitch " + brand.VersionLabel(m.Options.Version))
	subtitle := theme.style(theme.Muted).Render("safe Antigravity account control")
	line1 := title + "  " + subtitle

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
	accountChip := chip(theme, "ACCOUNT", active, theme.Accent)
	refreshChip := chip(theme, "REFRESH", refresh, theme.Info)
	thresholdChip := chip(theme, "AUTO", fmt.Sprintf("%d%%", m.Options.AutoThreshold), theme.Warning)
	line2 := lipgloss.JoinHorizontal(lipgloss.Center, accountChip, "  ", refreshChip, "  ", thresholdChip)

	if m.EditingRefresh {
		line2 = theme.style(theme.Info).Bold(true).Render("Refresh timer: "+m.RefreshInput+"s_") + theme.style(theme.Muted).Render("  Enter save · Esc cancel")
	}
	if m.EditingThreshold {
		line2 = theme.style(theme.Warning).Bold(true).Render("Auto-switch threshold: "+m.ThresholdInput+"%_") + theme.style(theme.Muted).Render("  Enter save · Esc cancel")
	}
	return trimWidth(line1, width) + "\n" + trimWidth(line2, width) + "\n" + trimWidth(status, width)
}

func chip(theme tuiTheme, label, value string, c color.Color) string {
	labelStyle := theme.style(theme.Muted).Bold(true)
	valueStyle := theme.style(c).Bold(true)
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
}

func renderCommandPanel(m Model, width, height int) string {
	return renderCommandPanelWithTheme(m, width, height, currentTheme())
}

func renderCommandPanelWithTheme(m Model, width, height int, theme tuiTheme) string {
	lines := make([]string, 0, len(dashboardCommands))
	for index, command := range dashboardCommands {
		description := command.Description
		switch command.Action {
		case actionAutoRefresh:
			if m.Options.AutoRefresh <= 0 {
				description = "Disabled · Enter seconds or 0"
			} else {
				description = "Every " + compactDuration(m.Options.AutoRefresh) + " · Enter to change"
			}
		case actionAutoSwitch:
			if m.Decision.Switch {
				description = "Recommended: " + m.Decision.Selected.Profile
			} else {
				description = m.Decision.Reason
			}
		case actionAutoThreshold:
			description = fmt.Sprintf("Current %d%% · Enter to change", m.Options.AutoThreshold)
		}
		shortcut := ""
		if command.Shortcut != "" {
			shortcut = theme.style(theme.Accent).Bold(true).Render("[" + command.Shortcut + "]") + " "
		}
		iconStyle := theme.style(theme.Accent)
		if command.Action == actionHotReload || command.Action == actionAutoSwitch {
			iconStyle = theme.style(theme.Success).Bold(true)
		}
		line := fmt.Sprintf("%s %s%-16s %s", iconStyle.Render(command.Icon), shortcut, command.Label, theme.style(theme.Muted).Render(description))
		if m.Focus == focusCommands && index == m.SelectedCommand {
			line = selectedLine(theme, trimWidth(stripANSI(line), width-6), width-6)
		}
		lines = append(lines, line)
	}
	focused := m.Focus == focusCommands || m.Focus == focusRefreshInput || m.Focus == focusThresholdInput
	return renderBoxWithTheme("Actions", strings.Join(lines, "\n"), width, height, focused, theme)
}

func renderAccountPanel(m Model, width, height int) string {
	return renderAccountPanelWithTheme(m, width, height, currentTheme())
}

func renderAccountPanelWithTheme(m Model, width, height int, theme tuiTheme) string {
	cursor := ""
	if m.Searching {
		cursor = "_"
	}
	search := theme.style(theme.Info).Bold(true).Render("Search") + theme.style(theme.Muted).Render(": "+m.Search+cursor)
	lines := []string{search}
	items := m.filteredAccounts()
	now := time.Now()
	if len(items) == 0 {
		lines = append(lines, theme.style(theme.Muted).Render("No matching accounts"))
	}
	for index, item := range items {
		active := theme.style(theme.Muted).Render("○")
		if item.Active {
			active = theme.style(theme.Success).Bold(true).Render("●")
		}
		result := m.resultFor(item.ID)
		live, state := quotaStatus(result, now)
		badge := renderQuotaBadgeWithTheme(state, theme)
		score := ""
		if live {
			if remaining, ok := quota.MinimumKnownRemaining(result.Snapshot); ok {
				score = " " + theme.style(quotaThemeColor(theme, remaining)).Bold(true).Render(fmt.Sprintf("%d%%", remaining))
			}
		}
		recommended := ""
		if m.Decision.Switch && m.Decision.Selected.Profile == item.ID {
			recommended = " " + theme.style(theme.Warning).Bold(true).Render("★ BEST")
		}
		nameWidth := max(8, width-visibleWidth(badge)-visibleWidth(score)-visibleWidth(recommended)-12)
		primaryPlain := fmt.Sprintf("%s %s", stripANSI(active), padRight(item.ID, nameWidth))
		primary := active + " " + theme.style(theme.Text).Bold(item.Active).Render(padRight(trimWidth(item.ID, nameWidth), nameWidth)) + "  " + badge + score + recommended
		email := strings.TrimSpace(item.Email)
		if email == "" || email == item.ID {
			email = "No email metadata"
		}
		secondary := "  " + theme.style(theme.Muted).Render(trimWidth(email, width-8))
		if m.Focus == focusAccounts && index == m.SelectedAccount {
			primary = selectedLine(theme, primaryPlain+"  "+state+stripANSI(score)+stripANSI(recommended), width-6)
			secondary = selectedLine(theme, "  "+email, width-6)
		}
		lines = append(lines, primary, secondary)
	}
	focused := m.Focus == focusAccounts || m.Focus == focusSearch
	return renderBoxWithTheme("Accounts", strings.Join(lines, "\n"), width, height, focused, theme)
}

func selectedLine(theme tuiTheme, text string, width int) string {
	style := lipgloss.NewStyle().Bold(true)
	if theme.ColorsEnabled {
		style = style.Foreground(theme.SelectedText).Background(theme.SelectedBG)
	} else {
		style = style.Reverse(true)
	}
	return style.Width(max(1, width)).Render(trimWidth(text, max(1, width)))
}

func quotaStatus(result quota.Result, now time.Time) (bool, string) {
	if result.Err != nil {
		return false, "ERROR"
	}
	snapshot := result.Snapshot
	if snapshot.Source == "cache-stale" || strings.TrimSpace(snapshot.Metadata["warning"]) != "" {
		return false, "STALE"
	}
	if snapshot.Source != "google-cloud-code" || snapshot.FetchedAt.IsZero() {
		return false, "UNKNOWN"
	}
	if now.Sub(snapshot.FetchedAt) > quota.DefaultAutoSwitchMaxAge {
		return false, "OLD"
	}
	if _, ok := quota.MinimumKnownRemaining(snapshot); !ok {
		return false, "UNKNOWN"
	}
	return true, "LIVE"
}

func renderQuotaBadge(state string) string {
	return renderQuotaBadgeWithTheme(state, currentTheme())
}

func renderQuotaBadgeWithTheme(state string, theme tuiTheme) string {
	var c color.Color
	switch state {
	case "LIVE":
		c = theme.Success
	case "STALE", "OLD":
		c = theme.Warning
	case "ERROR":
		c = theme.Danger
	default:
		c = theme.Muted
	}
	return theme.style(c).Bold(true).Render("[" + state + "]")
}

func renderQuotaPanel(m Model, width, height int) string {
	return renderQuotaPanelWithTheme(m, width, height, currentTheme())
}

func renderQuotaPanelWithTheme(m Model, width, height int, theme tuiTheme) string {
	profile, ok := m.selectedProfile()
	if !ok {
		return renderBoxWithTheme("Model quota", theme.style(theme.Muted).Render("Select an account to inspect model quota."), width, height, false, theme)
	}
	result := m.resultFor(profile)
	if result.Err != nil {
		return renderBoxWithTheme("Model quota · "+profile, theme.style(theme.Danger).Bold(true).Render("Unavailable: "+result.Err.Error()), width, height, false, theme)
	}
	live, state := quotaStatus(result, time.Now())
	models := quota.SortedModels(result.Snapshot)
	if len(models) == 0 {
		return renderBoxWithTheme("Model quota · "+profile, theme.style(theme.Muted).Render("No model quota returned."), width, height, false, theme)
	}

	entries := make([]string, 0, len(models)+2)
	if !live {
		entries = append(entries, theme.style(theme.Warning).Bold(true).Render(state+" quota is display-only and cannot trigger auto-switch."))
		if warning := strings.TrimSpace(result.Snapshot.Metadata["warning"]); warning != "" {
			entries = append(entries, theme.style(theme.Muted).Render(trimWidth(warning, width-8)))
		}
	}
	now := time.Now()
	for _, model := range models {
		remaining := "unknown"
		bar := theme.style(theme.Muted).Render("[··········]")
		if model.Remaining >= 0 {
			remaining = fmt.Sprintf("%3d%%", model.Remaining)
			if live {
				c := quotaThemeColor(theme, model.Remaining)
				bar = theme.style(c).Render(compactBar(model.Remaining, 10))
				remaining = theme.style(c).Bold(true).Render(remaining)
			} else {
				bar = theme.style(theme.Muted).Render(compactBar(model.Remaining, 10))
				remaining = theme.style(theme.Muted).Render(remaining)
			}
		}
		variants := ""
		if model.Variants > 1 {
			variants = fmt.Sprintf(" ×%d", model.Variants)
		}
		resetText := ""
		if duration := quota.ResetIn(model.ResetAt, now); duration > 0 {
			resetText = theme.style(theme.Muted).Render(" · reset " + compactDuration(duration))
		}
		name := theme.style(theme.Text).Render(padRight(trimWidth(model.Name, 28), 28))
		entries = append(entries, fmt.Sprintf("%s %s %s%s%s", name, bar, remaining, variants, resetText))
	}
	body := renderGrid(entries, width-6)
	rows := len(entries)
	if width >= 104 {
		rows = (len(entries) + 1) / 2
	}
	height = min(height, max(5, rows+2))
	age := "unknown age"
	if !result.Snapshot.FetchedAt.IsZero() {
		age = compactDuration(maxDuration(0, now.Sub(result.Snapshot.FetchedAt))) + " ago"
	}
	title := fmt.Sprintf("Model quota · %s · %s · %s", profile, state, age)
	return renderBoxWithTheme(title, body, width, height, false, theme)
}

func quotaThemeColor(theme tuiTheme, remaining int) color.Color {
	switch {
	case remaining <= 20:
		return theme.Danger
	case remaining <= 50:
		return theme.Warning
	default:
		return theme.Success
	}
}

func renderConfirmationPanel(m Model, width, height int) string {
	return renderConfirmationPanelWithTheme(m, width, height, currentTheme())
}

func renderConfirmationPanelWithTheme(m Model, width, height int, theme tuiTheme) string {
	title := theme.style(theme.Warning).Bold(true).Render("⚠ " + m.ConfirmTitle)
	body := title + "\n\n" + theme.style(theme.Text).Render(m.ConfirmBody) + "\n\n" +
		theme.style(theme.Success).Bold(true).Render("Enter / Y  confirm") + "     " +
		theme.style(theme.Danger).Bold(true).Render("Esc / N  cancel")
	return renderBoxWithTheme("Safety confirmation", body, width, height, true, theme)
}

func renderGrid(entries []string, width int) string {
	if width < 104 {
		return strings.Join(entries, "\n")
	}
	columnWidth := width/2 - 1
	rows := (len(entries) + 1) / 2
	lines := make([]string, 0, rows)
	for index := 0; index < rows; index++ {
		left := entries[index]
		right := ""
		if index+rows < len(entries) {
			right = entries[index+rows]
		}
		lines = append(lines, padRight(trimWidth(left, columnWidth), columnWidth)+"  "+trimWidth(right, columnWidth))
	}
	return strings.Join(lines, "\n")
}

func renderHelp(m Model, width int) string {
	return renderHelpWithTheme(m, width, currentTheme())
}

func renderHelpWithTheme(m Model, width int, theme tuiTheme) string {
	text := "Tab panels  ↑/↓ move  Enter run  s hot-switch  / search  r refresh  a auto  A threshold  p previous  d doctor  q quit"
	if m.confirming() {
		text = "Enter / Y confirm · Esc / N cancel · q quit"
	} else if m.EditingRefresh {
		text = "Type seconds · 0 disables · Enter save · Esc cancel"
	} else if m.EditingThreshold {
		text = "Type threshold 0–100 · Enter save · Esc cancel"
	} else if m.Searching {
		text = "Type to search · Enter apply · Ctrl-U clear · Esc cancel"
	}
	return theme.style(theme.Muted).Render(trimWidth(text, width))
}

func renderBox(title, body string, width, height int, focused bool) string {
	return renderBoxWithTheme(title, body, width, height, focused, currentTheme())
}

func renderBoxWithTheme(title, body string, width, height int, focused bool, theme tuiTheme) string {
	if width < 10 {
		return body
	}
	borderColor := theme.Border
	if focused {
		borderColor = theme.Accent
		title = "● " + title
	}
	titleStyle := theme.style(borderColor).Bold(true)
	body = truncateLines(body, max(1, height-2), max(1, width-6))
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(max(1, width-4)).
		Height(max(1, height-2))
	if theme.ColorsEnabled {
		style = style.Foreground(theme.Text).BorderForeground(borderColor)
	}
	content := titleStyle.Render(title) + "\n" + body
	return style.Render(content)
}

func compactBar(value, width int) string {
	if value < 0 {
		return "[" + strings.Repeat("·", width) + "]"
	}
	if value > 100 {
		value = 100
	}
	filled := value * width / 100
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func compactDuration(value time.Duration) string {
	if value >= 24*time.Hour {
		return fmt.Sprintf("%dd %dh", int(value/(24*time.Hour)), int(value%(24*time.Hour)/time.Hour))
	}
	if value >= time.Hour {
		minutes := int(value % time.Hour / time.Minute)
		if minutes == 0 {
			return fmt.Sprintf("%dh", int(value/time.Hour))
		}
		return fmt.Sprintf("%dh %dm", int(value/time.Hour), minutes)
	}
	if value >= time.Minute {
		return fmt.Sprintf("%dm", max(1, int(value/time.Minute)))
	}
	return fmt.Sprintf("%ds", max(1, int(value/time.Second)))
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func truncateLines(value string, maxLines, width int) string {
	lines := strings.Split(value, "\n")
	if len(lines) > maxLines {
		lines = append(lines[:maxLines-1], "...")
	}
	for index := range lines {
		lines[index] = trimWidth(lines[index], width)
	}
	return strings.Join(lines, "\n")
}

func trimWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	plain := stripANSI(sanitizeTerminalText(value))
	if lipgloss.Width(plain) <= width {
		return sanitizeTerminalText(value)
	}
	runes := []rune(plain)
	if width == 1 {
		return string(runes[:1])
	}
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func padRight(value string, width int) string {
	value = sanitizeTerminalText(value)
	missing := width - visibleWidth(value)
	if missing <= 0 {
		return value
	}
	return value + strings.Repeat(" ", missing)
}

func visibleWidth(value string) int {
	return lipgloss.Width(stripANSI(sanitizeTerminalText(value)))
}

func sanitizeTerminalText(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\t':
			return ' '
		case '\r':
			return -1
		}
		if r == '\n' || r == '\033' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, value)
}

func stripANSI(value string) string {
	var output strings.Builder
	inEscape := false
	for _, r := range value {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		output.WriteRune(r)
	}
	return output.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
