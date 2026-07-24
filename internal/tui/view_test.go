package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ibrahim-wael/agswitch/internal/account"
)

func TestSanitizeTerminalTextRemovesTabsAndControls(t *testing.T) {
	t.Parallel()
	actual := sanitizeTerminalText("Model\tQuota\r\x00")
	if actual != "Model Quota" {
		t.Fatalf("sanitizeTerminalText() = %q; want %q", actual, "Model Quota")
	}
}

func TestRenderHeaderNormalizesVersionAndShowsRefresh(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	model := Model{
		Options: Options{
			Version:       "vv1.0.0",
			AutoThreshold: 25,
			AutoRefresh:   30 * time.Second,
		},
		Status: "Ready",
	}
	header := stripANSI(renderHeader(160, model))
	for _, expected := range []string{"AGSwitch v1.0.0", "Refresh: every 30s", "Auto switch: 25%"} {
		if !strings.Contains(header, expected) {
			t.Fatalf("header does not contain %q: %q", expected, header)
		}
	}
	if strings.Contains(header, "vv1.0.0") {
		t.Fatalf("header contains duplicate version prefix: %q", header)
	}
}

func TestAccountPanelShowsEmailOnOwnLine(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	model := Model{
		Accounts: []account.Account{{ID: "work", Email: "ibrahim@example.com", Active: true}},
		Focus:    focusAccounts,
	}
	panel := stripANSI(renderAccountPanel(model, 70, 7))
	if !strings.Contains(panel, "ibrahim@example.com") {
		t.Fatalf("account panel omitted email: %q", panel)
	}
}

func TestViewUsesAlternateScreen(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	model := Model{Width: 100, Height: 30, Status: "Ready"}
	view := model.View()
	if !view.AltScreen {
		t.Fatal("dashboard should use the alternate screen buffer")
	}
}

func TestAutoSwitchOperationDoesNotQuitDashboard(t *testing.T) {
	model := Model{Options: Options{ExitAfterSwitch: true}}
	updated, command := model.Update(operationMsg{action: actionAutoSwitch, profile: "work"})
	if command == nil {
		t.Fatal("auto-switch completion should reload dashboard data")
	}
	if _, ok := updated.(Model); !ok {
		t.Fatalf("unexpected updated model type %T", updated)
	}
	if command == tea.Quit {
		t.Fatal("auto-switch should not quit the dashboard")
	}
}

func TestRenderGridUsesOneColumnOnNarrowTerminals(t *testing.T) {
	t.Parallel()
	entries := []string{"one", "two", "three", "four"}
	narrow := renderGrid(entries, 80)
	if narrow != strings.Join(entries, "\n") {
		t.Fatalf("narrow grid should use one column: %q", narrow)
	}
	wide := renderGrid(entries, 120)
	if !strings.Contains(wide, "one") || !strings.Contains(wide, "three") {
		t.Fatalf("wide grid omitted entries: %q", wide)
	}
}

func TestCompactDurationSupportsSecondsAndHours(t *testing.T) {
	t.Parallel()
	cases := map[time.Duration]string{
		30 * time.Second:             "30s",
		5 * time.Minute:              "5m",
		5*time.Hour + 12*time.Minute: "5h 12m",
		26 * time.Hour:               "1d 2h",
	}
	for input, expected := range cases {
		if actual := compactDuration(input); actual != expected {
			t.Fatalf("compactDuration(%s) = %q; want %q", input, actual, expected)
		}
	}
}
