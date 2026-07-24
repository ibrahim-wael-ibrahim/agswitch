package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/ibrahim-wael/agswitch/internal/brand"
	"github.com/ibrahim-wael/agswitch/internal/quota"
)

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	reverse = "\033[7m"
)

func (m Model) View() tea.View {
	width := m.Width
	if width <= 0 {
		width = 100
	}
	height := m.Height
	if height <= 0 {
		height = 32
	}
	contentWidth := max(40, width-2)

	var output strings.Builder
	output.WriteString(renderHeader(contentWidth, m))
	output.WriteByte('\n')

	commandHeight := min(max(len(dashboardCommands)+2, 8), 13)
	accountHeight := min(max(len(m.filteredAccounts())+3, 8), 13)
	topHeight := max(commandHeight, accountHeight)
	if contentWidth >= 92 {
		leftWidth := contentWidth/2 - 1
		rightWidth := contentWidth - leftWidth - 1
		output.WriteString(joinColumns(
			renderCommandPanel(m, leftWidth, topHeight),
			renderAccountPanel(m, rightWidth, topHeight),
			leftWidth,
			rightWidth,
		))
	} else {
		output.WriteString(renderCommandPanel(m, contentWidth, commandHeight))
		output.WriteByte('\n')
		output.WriteString(renderAccountPanel(m, contentWidth, accountHeight))
		topHeight = commandHeight + accountHeight + 1
	}

	output.WriteByte('\n')
	reserved := topHeight + 7
	if strings.TrimSpace(m.Details) != "" {
		reserved += 7
	}
	quotaHeight := max(7, height-reserved)
	output.WriteString(renderQuotaPanel(m, contentWidth, quotaHeight))
	if strings.TrimSpace(m.Details) != "" {
		output.WriteByte('\n')
		output.WriteString(renderBox("Details", truncateLines(m.Details, 5, contentWidth-4), contentWidth, 7, false))
	}
	output.WriteByte('\n')
	output.WriteString(renderHelp(m, contentWidth))
	return tea.NewView(output.String())
}

func renderHeader(width int, m Model) string {
	version := brand.VersionLabel(m.Options.Version)
	refresh := "manual"
	if m.Options.AutoRefresh > 0 {
		refresh = fmt.Sprintf("every %s", compactDuration(m.Options.AutoRefresh))
	}
	busy := ""
	if m.Busy {
		busy = " · working"
	}
	line1 := bold + "AGSwitch " + version + reset + "  " + dim + "by " + brand.Author + " · " + brand.Repository + reset
	line2 := fmt.Sprintf("%sStatus:%s %s%s  %sAuto refresh:%s %s  %sThreshold:%s %d%%",
		bold, reset, sanitizeTerminalText(m.Status), busy,
		bold, reset, refresh,
		bold, reset, m.Options.AutoThreshold,
	)
	if m.EditingRefresh {
		line2 = fmt.Sprintf("%sAuto refresh seconds:%s %s_  %s(0 disables · Enter save · Esc cancel)%s",
			bold, reset, m.RefreshInput, dim, reset)
	}
	return trimWidth(line1, width) + "\n" + trimWidth(line2, width)
}

func renderCommandPanel(m Model, width, height int) string {
	lines := make([]string, 0, len(dashboardCommands))
	for index, command := range dashboardCommands {
		description := command.Description
		if command.Action == actionAutoRefresh {
			if m.Options.AutoRefresh <= 0 {
				description = "Disabled · Enter seconds or 0"
			} else {
				description = "Every " + compactDuration(m.Options.AutoRefresh) + " · Enter to change"
			}
		}
		line := fmt.Sprintf("  %-17s %s", command.Label, description)
		if m.Focus == focusCommands && index == m.SelectedCommand {
			line = reverse + trimWidth(line, width-4) + reset
		}
		lines = append(lines, line)
	}
	return renderBox("Program & Commands", strings.Join(lines, "\n"), width, height, m.Focus == focusCommands || m.Focus == focusRefreshInput)
}

func renderAccountPanel(m Model, width, height int) string {
	query := m.Search
	cursor := ""
	if m.Searching {
		cursor = "_"
	}
	lines := []string{"Search: " + query + cursor}
	items := m.filteredAccounts()
	if len(items) == 0 {
		lines = append(lines, dim+"No matching accounts"+reset)
	}
	for index, item := range items {
		active := " "
		if item.Active {
			active = "●"
		}
		score := "unknown"
		if remaining, ok := quota.MinimumKnownRemaining(m.resultFor(item.ID).Snapshot); ok {
			score = fmt.Sprintf("%d%%", remaining)
		}
		recommended := ""
		if m.Decision.Selected.Profile == item.ID {
			recommended = " · best"
		}
		email := item.Email
		if strings.TrimSpace(email) == "" || email == item.ID {
			email = ""
		}
		available := max(8, width-20)
		identity := strings.TrimSpace(item.ID + "  " + email)
		text := fmt.Sprintf("%s %-*s %7s%s", active, available, trimWidth(identity, available), score, recommended)
		if m.Focus == focusAccounts && index == m.SelectedAccount {
			text = reverse + trimWidth(text, width-4) + reset
		}
		lines = append(lines, text)
	}
	return renderBox("Search & Accounts", strings.Join(lines, "\n"), width, height, m.Focus == focusAccounts || m.Focus == focusSearch)
}

func renderQuotaPanel(m Model, width, height int) string {
	profile, ok := m.selectedProfile()
	if !ok {
		return renderBox("Model Quota", dim+"Select an account to inspect model quota."+reset, width, height, false)
	}
	result := m.resultFor(profile)
	if result.Err != nil {
		return renderBox("Model Quota · "+profile, "Unavailable: "+result.Err.Error(), width, height, false)
	}
	models := quota.SortedModels(result.Snapshot)
	if len(models) == 0 {
		return renderBox("Model Quota · "+profile, dim+"No model quota returned."+reset, width, height, false)
	}
	entries := make([]string, 0, len(models))
	now := time.Now()
	for _, model := range models {
		remaining := "unknown"
		bar := "[··········]"
		if model.Remaining >= 0 {
			remaining = fmt.Sprintf("%3d%%", model.Remaining)
			bar = compactBar(model.Remaining, 10)
		}
		variants := ""
		if model.Variants > 1 {
			variants = fmt.Sprintf(" ×%d", model.Variants)
		}
		resetText := ""
		if duration := quota.ResetIn(model.ResetAt, now); duration > 0 {
			resetText = " · " + compactDuration(duration)
		}
		entries = append(entries, fmt.Sprintf("%-30s %s %s%s%s", trimWidth(model.Name, 30), bar, remaining, variants, resetText))
	}
	body := renderGrid(entries, width-4)
	title := fmt.Sprintf("Model Quota · %s · %s", profile, result.Snapshot.Source)
	return renderBox(title, body, width, height, false)
}

func renderGrid(entries []string, width int) string {
	if width < 100 {
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
	text := "Tab panels  ↑/↓ move  Enter select/run  / search  r refresh  R auto-refresh  a auto-switch  p previous  d doctor  q quit"
	if m.EditingRefresh {
		text = "Type seconds · 0 disables · Enter save · Esc cancel"
	} else if m.Searching {
		text = "Type to search · Enter apply · Ctrl-U clear · Esc cancel"
	}
	return dim + trimWidth(text, width) + reset
}

func renderBox(title, body string, width, height int, focused bool) string {
	if width < 10 {
		return body
	}
	marker := ""
	if focused {
		marker = "● "
	}
	title = marker + sanitizeTerminalText(title)
	top := "┌─ " + trimWidth(title, width-6) + " " + strings.Repeat("─", max(0, width-visibleWidth(title)-5)) + "┐"
	bottom := "└" + strings.Repeat("─", max(0, width-2)) + "┘"
	bodyLines := strings.Split(sanitizeTerminalText(body), "\n")
	maxBody := max(1, height-2)
	if len(bodyLines) > maxBody {
		bodyLines = bodyLines[:maxBody]
	}
	for len(bodyLines) < maxBody {
		bodyLines = append(bodyLines, "")
	}
	lines := []string{top}
	for _, line := range bodyLines {
		line = trimWidth(line, width-4)
		lines = append(lines, "│ "+padRight(line, width-4)+" │")
	}
	lines = append(lines, bottom)
	return strings.Join(lines, "\n")
}

func joinColumns(left, right string, leftWidth, rightWidth int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	rows := max(len(leftLines), len(rightLines))
	lines := make([]string, 0, rows)
	for index := 0; index < rows; index++ {
		leftLine, rightLine := "", ""
		if index < len(leftLines) {
			leftLine = leftLines[index]
		}
		if index < len(rightLines) {
			rightLine = rightLines[index]
		}
		lines = append(lines, padRight(leftLine, leftWidth)+" "+padRight(rightLine, rightWidth))
	}
	return strings.Join(lines, "\n")
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
		minutes := int(value%time.Hour/time.Minute)
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
	if len([]rune(plain)) <= width {
		return sanitizeTerminalText(value)
	}
	runes := []rune(plain)
	if width == 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
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
	return len([]rune(stripANSI(sanitizeTerminalText(value))))
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
