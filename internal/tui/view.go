package tui

import (
	"fmt"
	"os"
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
	cyan    = "\033[36m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	red     = "\033[31m"
	blue    = "\033[34m"
)

func (m Model) View() tea.View {
	w, h := m.Width, m.Height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 32
	}
	w = max(40, w-2)
	var out strings.Builder
	out.WriteString(renderHeader(w, m))
	out.WriteByte('\n')
	if m.confirming() {
		out.WriteString(renderConfirmationPanel(m, w, min(max(8, h-6), 12)))
		out.WriteByte('\n')
		out.WriteString(renderHelp(m, w))
		v := tea.NewView(out.String())
		v.AltScreen = true
		return v
	}
	ch := min(max(len(dashboardCommands)+2, 10), 16)
	ah := min(max(len(m.filteredAccounts())*2+3, 10), 16)
	top := max(ch, ah)
	if w >= 92 {
		lw := w/2 - 1
		rw := w - lw - 1
		out.WriteString(joinColumns(renderCommandPanel(m, lw, top), renderAccountPanel(m, rw, top), lw, rw))
	} else {
		out.WriteString(renderCommandPanel(m, w, ch))
		out.WriteByte('\n')
		out.WriteString(renderAccountPanel(m, w, ah))
		top = ch + ah + 1
	}
	out.WriteByte('\n')
	reserved := top + 7
	if strings.TrimSpace(m.Details) != "" {
		reserved += 6
	}
	out.WriteString(renderQuotaPanel(m, w, min(max(7, h-reserved), 14)))
	if strings.TrimSpace(m.Details) != "" {
		out.WriteByte('\n')
		out.WriteString(renderBox("Details", truncateLines(m.Details, 4, w-4), w, 6, false))
	}
	out.WriteByte('\n')
	out.WriteString(renderHelp(m, w))
	v := tea.NewView(out.String())
	v.AltScreen = true
	return v
}

func renderHeader(width int, m Model) string {
	refresh := "manual"
	if m.Options.AutoRefresh > 0 {
		refresh = "every " + compactDuration(m.Options.AutoRefresh)
	}
	statusColor, busy := green, ""
	if m.Busy {
		statusColor, busy = yellow, " · working"
	}
	low := strings.ToLower(m.Status)
	if strings.Contains(low, "failed") || strings.Contains(low, "error") {
		statusColor = red
	}
	active := m.activeProfile()
	if active == "" {
		active = "unknown"
	}
	line1 := paint(cyan, bold+"AGSwitch "+brand.VersionLabel(m.Options.Version)+reset) + "  " + dim + "by " + brand.Author + " · " + brand.Repository + reset
	line2 := fmt.Sprintf("%sAccount:%s %s  %sStatus:%s %s%s  %sRefresh:%s %s  %sAuto switch:%s %d%%", bold, reset, active, bold, reset, paint(statusColor, m.Status), busy, bold, reset, refresh, bold, reset, m.Options.AutoThreshold)
	if m.EditingRefresh {
		line2 = fmt.Sprintf("%sAuto refresh seconds:%s %s_  %s(0 disables · Enter save · Esc cancel)%s", bold, reset, m.RefreshInput, dim, reset)
	}
	if m.EditingThreshold {
		line2 = fmt.Sprintf("%sAuto-switch threshold:%s %s%%_  %s(0–100 · Enter save · Esc cancel)%s", bold, reset, m.ThresholdInput, dim, reset)
	}
	return trimWidth(line1, width) + "\n" + trimWidth(line2, width)
}

func renderCommandPanel(m Model, width, height int) string {
	lines := make([]string, 0, len(dashboardCommands))
	for i, c := range dashboardCommands {
		d := c.Description
		switch c.Action {
		case actionAutoRefresh:
			if m.Options.AutoRefresh <= 0 {
				d = "Disabled · Enter seconds or 0"
			} else {
				d = "Every " + compactDuration(m.Options.AutoRefresh) + " · Enter to change"
			}
		case actionAutoSwitch:
			if m.Decision.Switch {
				d = "Recommended: " + m.Decision.Selected.Profile + " · confirmation required"
			} else {
				d = "No switch needed · " + m.Decision.Reason
			}
		case actionAutoThreshold:
			d = fmt.Sprintf("Current %d%% · Enter to change", m.Options.AutoThreshold)
		}
		shortcut := ""
		if c.Shortcut != "" {
			shortcut = " [" + c.Shortcut + "]"
		}
		line := fmt.Sprintf("  %s %-17s%s  %s", c.Icon, c.Label, shortcut, d)
		if m.Focus == focusCommands && i == m.SelectedCommand {
			line = reverse + trimWidth(line, width-4) + reset
		}
		lines = append(lines, line)
	}
	focused := m.Focus == focusCommands || m.Focus == focusRefreshInput || m.Focus == focusThresholdInput
	return renderBox("Actions", strings.Join(lines, "\n"), width, height, focused)
}

func renderAccountPanel(m Model, width, height int) string {
	cursor := ""
	if m.Searching {
		cursor = "_"
	}
	lines := []string{paint(blue, "Search:") + " " + m.Search + cursor}
	items := m.filteredAccounts()
	now := time.Now()
	if len(items) == 0 {
		lines = append(lines, dim+"No matching accounts"+reset)
	}
	for i, item := range items {
		active := "○"
		if item.Active {
			active = paint(green, "●")
		}
		result := m.resultFor(item.ID)
		live, state := quotaStatus(result, now)
		badge := renderQuotaBadge(state)
		score := ""
		if live {
			if n, ok := quota.MinimumKnownRemaining(result.Snapshot); ok {
				score = " " + paint(quotaColor(n), fmt.Sprintf("%d%%", n))
			}
		}
		rec := ""
		if m.Decision.Switch && m.Decision.Selected.Profile == item.ID {
			rec = "  " + paint(yellow, "★ recommended")
		}
		pw := max(8, width-visibleWidth(badge)-visibleWidth(score)-visibleWidth(rec)-10)
		primary := fmt.Sprintf("%s %s  %s%s%s", active, padRight(trimWidth(item.ID, pw), pw), badge, score, rec)
		email := strings.TrimSpace(item.Email)
		if email == "" || email == item.ID {
			email = "No email metadata"
		}
		secondary := "    " + dim + trimWidth(email, width-8) + reset
		if m.Focus == focusAccounts && i == m.SelectedAccount {
			primary = reverse + trimWidth(primary, width-4) + reset
			secondary = reverse + trimWidth("    "+email, width-4) + reset
		}
		lines = append(lines, primary, secondary)
	}
	return renderBox("Accounts", strings.Join(lines, "\n"), width, height, m.Focus == focusAccounts || m.Focus == focusSearch)
}

func quotaStatus(result quota.Result, now time.Time) (bool, string) {
	if result.Err != nil {
		return false, "ERROR"
	}
	s := result.Snapshot
	if s.Source == "cache-stale" || strings.TrimSpace(s.Metadata["warning"]) != "" {
		return false, "STALE"
	}
	if s.Source != "google-cloud-code" || s.FetchedAt.IsZero() {
		return false, "UNKNOWN"
	}
	if now.Sub(s.FetchedAt) > quota.DefaultAutoSwitchMaxAge {
		return false, "OLD"
	}
	if _, ok := quota.MinimumKnownRemaining(s); !ok {
		return false, "UNKNOWN"
	}
	return true, "LIVE"
}

func renderQuotaBadge(state string) string {
	switch state {
	case "LIVE":
		return paint(green, "[LIVE]")
	case "STALE":
		return paint(yellow, "[STALE]")
	case "OLD":
		return paint(yellow, "[OLD]")
	case "ERROR":
		return paint(red, "[ERROR]")
	}
	return dim + "[UNKNOWN]" + reset
}

func renderQuotaPanel(m Model, width, height int) string {
	profile, ok := m.selectedProfile()
	if !ok {
		return renderBox("Model quota", dim+"Select an account to inspect model quota."+reset, width, height, false)
	}
	result := m.resultFor(profile)
	if result.Err != nil {
		return renderBox("Model quota · "+profile, paint(red, "Unavailable: "+result.Err.Error()), width, height, false)
	}
	live, state := quotaStatus(result, time.Now())
	models := quota.SortedModels(result.Snapshot)
	if len(models) == 0 {
		return renderBox("Model quota · "+profile, dim+"No model quota returned."+reset, width, height, false)
	}
	entries := make([]string, 0, len(models)+2)
	if !live {
		entries = append(entries, paint(yellow, state+" quota is display-only and cannot trigger auto-switch."))
		if w := strings.TrimSpace(result.Snapshot.Metadata["warning"]); w != "" {
			entries = append(entries, dim+trimWidth(w, width-6)+reset)
		}
	}
	now := time.Now()
	for _, model := range models {
		remaining := "unknown"
		bar := dim + "[··········]" + reset
		if model.Remaining >= 0 {
			remaining = fmt.Sprintf("%3d%%", model.Remaining)
			if live {
				bar = paint(quotaColor(model.Remaining), compactBar(model.Remaining, 10))
				remaining = paint(quotaColor(model.Remaining), remaining)
			} else {
				bar = dim + compactBar(model.Remaining, 10) + reset
				remaining = dim + remaining + reset
			}
		}
		variants := ""
		if model.Variants > 1 {
			variants = fmt.Sprintf(" ×%d", model.Variants)
		}
		resetText := ""
		if d := quota.ResetIn(model.ResetAt, now); d > 0 {
			resetText = " · reset " + compactDuration(d)
		}
		entries = append(entries, fmt.Sprintf("%-30s %s %s%s%s", trimWidth(model.Name, 30), bar, remaining, variants, resetText))
	}
	body := renderGrid(entries, width-4)
	rows := len(entries)
	if width >= 100 {
		rows = (len(entries) + 1) / 2
	}
	height = min(height, max(5, rows+2))
	age := "unknown age"
	if !result.Snapshot.FetchedAt.IsZero() {
		age = compactDuration(maxDuration(0, now.Sub(result.Snapshot.FetchedAt))) + " ago"
	}
	return renderBox(fmt.Sprintf("Model quota · %s · %s · %s", profile, state, age), body, width, height, false)
}

func renderConfirmationPanel(m Model, width, height int) string {
	body := paint(yellow, bold+m.ConfirmTitle+reset) + "\n\n" + m.ConfirmBody + "\n\n" + paint(green, "Enter / Y") + " continue    " + paint(red, "Esc / N") + " cancel"
	return renderBox("Safety confirmation", body, width, height, true)
}

func renderGrid(entries []string, width int) string {
	if width < 100 {
		return strings.Join(entries, "\n")
	}
	cw := width/2 - 1
	rows := (len(entries) + 1) / 2
	lines := make([]string, 0, rows)
	for i := 0; i < rows; i++ {
		left := entries[i]
		right := ""
		if i+rows < len(entries) {
			right = entries[i+rows]
		}
		lines = append(lines, padRight(trimWidth(left, cw), cw)+"  "+trimWidth(right, cw))
	}
	return strings.Join(lines, "\n")
}

func renderHelp(m Model, width int) string {
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
	return dim + trimWidth(text, width) + reset
}

func renderBox(title, body string, width, height int, focused bool) string {
	if width < 10 {
		return body
	}
	marker := ""
	if focused {
		marker = paint(cyan, "● ")
	}
	title = marker + sanitizeTerminalText(title)
	top := "┌─ " + trimWidth(title, width-6) + " " + strings.Repeat("─", max(0, width-visibleWidth(title)-5)) + "┐"
	bottom := "└" + strings.Repeat("─", max(0, width-2)) + "┘"
	bl := strings.Split(sanitizeTerminalText(body), "\n")
	maxBody := max(1, height-2)
	if len(bl) > maxBody {
		bl = bl[:maxBody]
	}
	for len(bl) < maxBody {
		bl = append(bl, "")
	}
	lines := []string{top}
	for _, line := range bl {
		line = trimWidth(line, width-4)
		lines = append(lines, "│ "+padRight(line, width-4)+" │")
	}
	lines = append(lines, bottom)
	return strings.Join(lines, "\n")
}

func joinColumns(left, right string, lw, rw int) string {
	ll, rl := strings.Split(left, "\n"), strings.Split(right, "\n")
	rows := max(len(ll), len(rl))
	lines := make([]string, 0, rows)
	for i := 0; i < rows; i++ {
		l, r := "", ""
		if i < len(ll) {
			l = ll[i]
		}
		if i < len(rl) {
			r = rl[i]
		}
		lines = append(lines, padRight(l, lw)+" "+padRight(r, rw))
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
func compactDuration(v time.Duration) string {
	if v >= 24*time.Hour {
		return fmt.Sprintf("%dd %dh", int(v/(24*time.Hour)), int(v%(24*time.Hour)/time.Hour))
	}
	if v >= time.Hour {
		m := int(v % time.Hour / time.Minute)
		if m == 0 {
			return fmt.Sprintf("%dh", int(v/time.Hour))
		}
		return fmt.Sprintf("%dh %dm", int(v/time.Hour), m)
	}
	if v >= time.Minute {
		return fmt.Sprintf("%dm", max(1, int(v/time.Minute)))
	}
	return fmt.Sprintf("%ds", max(1, int(v/time.Second)))
}
func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
func truncateLines(v string, maxLines, width int) string {
	lines := strings.Split(v, "\n")
	if len(lines) > maxLines {
		lines = append(lines[:maxLines-1], "...")
	}
	for i := range lines {
		lines[i] = trimWidth(lines[i], width)
	}
	return strings.Join(lines, "\n")
}
func trimWidth(v string, width int) string {
	if width <= 0 {
		return ""
	}
	plain := stripANSI(sanitizeTerminalText(v))
	if len([]rune(plain)) <= width {
		return sanitizeTerminalText(v)
	}
	r := []rune(plain)
	if width == 1 {
		return string(r[:1])
	}
	return string(r[:width-1]) + "…"
}
func padRight(v string, width int) string {
	v = sanitizeTerminalText(v)
	missing := width - visibleWidth(v)
	if missing <= 0 {
		return v
	}
	return v + strings.Repeat(" ", missing)
}
func visibleWidth(v string) int { return len([]rune(stripANSI(sanitizeTerminalText(v)))) }
func sanitizeTerminalText(v string) string {
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
	}, v)
}
func stripANSI(v string) string {
	var out strings.Builder
	esc := false
	for _, r := range v {
		if r == '\033' {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
func colorsEnabled() bool {
	return os.Getenv("NO_COLOR") == "" && strings.ToLower(os.Getenv("TERM")) != "dumb"
}
func paint(color, v string) string {
	if !colorsEnabled() {
		return v
	}
	return color + v + reset
}
func quotaColor(v int) string {
	if v <= 20 {
		return red
	}
	if v <= 50 {
		return yellow
	}
	return green
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
