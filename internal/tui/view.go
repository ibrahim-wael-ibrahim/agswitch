package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"strings"
)

func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString("agswitch — Antigravity accounts\n\n")
	if len(m.Accounts) == 0 {
		b.WriteString("No profiles saved. Run: agswitch save <profile> or agswitch migrate\n\nq quit\n")
		return tea.NewView(b.String())
	}
	for i, a := range m.Accounts {
		c := "  "
		if i == m.Selected {
			c = "> "
		}
		active := " "
		if a.Active {
			active = "*"
		}
		fmt.Fprintf(&b, "%s%s %-20s %s\n", c, active, a.ID, a.Email)
	}
	b.WriteString("\nStatus: " + m.Status + "\n\n↑/↓ or j/k move   enter switch+launch   r refresh   q quit\n")
	return tea.NewView(b.String())
}
