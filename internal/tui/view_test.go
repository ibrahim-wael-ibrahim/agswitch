package tui

import (
	"strings"
	"testing"
	"time"
)

func TestSanitizeTerminalTextRemovesTabsAndControls(t *testing.T) {
	t.Parallel()
	actual := sanitizeTerminalText("Model\tQuota\r\x00")
	if actual != "Model Quota" {
		t.Fatalf("sanitizeTerminalText() = %q; want %q", actual, "Model Quota")
	}
}

func TestRenderHeaderNormalizesVersionAndShowsRefresh(t *testing.T) {
	t.Parallel()
	model := Model{
		Options: Options{
			Version:       "vv1.0.0",
			AutoThreshold: 25,
			AutoRefresh:   30 * time.Second,
		},
		Status: "Ready",
	}
	header := stripANSI(renderHeader(160, model))
	for _, expected := range []string{"AGSwitch v1.0.0", "Auto refresh: every 30s", "Threshold: 25%"} {
		if !strings.Contains(header, expected) {
			t.Fatalf("header does not contain %q: %q", expected, header)
		}
	}
	if strings.Contains(header, "vv1.0.0") {
		t.Fatalf("header contains duplicate version prefix: %q", header)
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
		30 * time.Second:          "30s",
		5 * time.Minute:           "5m",
		5*time.Hour + 12*time.Minute: "5h 12m",
		26 * time.Hour:            "1d 2h",
	}
	for input, expected := range cases {
		if actual := compactDuration(input); actual != expected {
			t.Fatalf("compactDuration(%s) = %q; want %q", input, actual, expected)
		}
	}
}
