package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/quota"
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
	for _, expected := range []string{"AGSwitch v1.0.0", "REFRESH: every 30s", "AUTO: 25%", "Ready"} {
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

func TestAccountPanelMarksStaleQuotaWithoutPresentingPercentage(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	model := Model{
		Accounts: []account.Account{{ID: "old", Email: "old@example.com"}},
		Results: []quota.Result{{
			Profile: "old",
			Snapshot: quota.Snapshot{
				Profile:   "old",
				Source:    "cache-stale",
				FetchedAt: time.Now().Add(-24 * time.Hour),
				Models: map[string]quota.ModelUsage{
					"model": {ID: "model", Name: "Model", Remaining: 100, Limit: 100},
				},
				Metadata: map[string]string{"warning": "refresh unavailable"},
			},
		}},
		Focus: focusAccounts,
	}
	panel := stripANSI(renderAccountPanel(model, 80, 8))
	if !strings.Contains(panel, "STALE") {
		t.Fatalf("account panel should mark stale quota: %q", panel)
	}
	if strings.Contains(panel, "100%") {
		t.Fatalf("stale account summary must not present quota as usable: %q", panel)
	}
}

func TestQuotaStatusRecognizesRecentLiveSnapshot(t *testing.T) {
	now := time.Now()
	result := quota.Result{
		Profile: "work",
		Snapshot: quota.Snapshot{
			Profile:   "work",
			Source:    "google-cloud-code",
			FetchedAt: now.Add(-30 * time.Second),
			Models: map[string]quota.ModelUsage{
				"model": {ID: "model", Name: "Model", Remaining: 75, Limit: 100},
			},
			Metadata: map[string]string{},
		},
	}
	live, state := quotaStatus(result, now)
	if !live || state != "LIVE" {
		t.Fatalf("quotaStatus() = %v, %q; want true, LIVE", live, state)
	}
}

func TestNoColorDisablesSemanticForegrounds(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	theme := currentTheme()
	if theme.ColorsEnabled {
		t.Fatal("NO_COLOR should disable semantic colors")
	}
	if strings.Contains(theme.style(theme.Success).Render("ok"), "\x1b[") {
		t.Fatal("NO_COLOR rendering should not contain ANSI colors")
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

func TestAutoSwitchOperationKeepsDashboardActive(t *testing.T) {
	model := Model{Options: Options{ExitAfterSwitch: true}}
	updated, command := model.Update(operationMsg{action: actionAutoSwitch, profile: "work"})
	if command == nil {
		t.Fatal("auto-switch completion should reload dashboard data")
	}
	updatedModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("unexpected updated model type %T", updated)
	}
	if !updatedModel.Busy {
		t.Fatal("auto-switch completion should keep the dashboard active while data reloads")
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
