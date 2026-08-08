package quota

import (
	"testing"
	"time"
)

func TestSummarizeWindowsSeparatesFiveHourAndWeekly(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	snapshot := Snapshot{Models: map[string]ModelUsage{
		"five-a": {Remaining: 72, ResetAt: now.Add(4 * time.Hour)},
		"five-b": {Remaining: 18, ResetAt: now.Add(4*time.Hour + 10*time.Minute)},
		"weekly": {Remaining: 64, ResetAt: now.Add(4 * 24 * time.Hour)},
	}}

	windows := SummarizeWindows(snapshot, now)
	if !windows.FiveHour.Available || windows.FiveHour.Remaining != 18 {
		t.Fatalf("five-hour window = %#v", windows.FiveHour)
	}
	if windows.FiveHour.Models != 2 {
		t.Fatalf("five-hour model count = %d; want 2", windows.FiveHour.Models)
	}
	if !windows.Weekly.Available || windows.Weekly.Remaining != 64 {
		t.Fatalf("weekly window = %#v", windows.Weekly)
	}
}

func TestSummarizeWindowsDoesNotInventWeeklyQuota(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	snapshot := Snapshot{Models: map[string]ModelUsage{
		"five": {Remaining: 55, ResetAt: now.Add(3 * time.Hour)},
	}}

	windows := SummarizeWindows(snapshot, now)
	if !windows.FiveHour.Available {
		t.Fatal("five-hour window should be available")
	}
	if windows.Weekly.Available {
		t.Fatalf("weekly window must stay unavailable without a provider long reset: %#v", windows.Weekly)
	}
}

func TestSummarizeWindowsMarksExhausted(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	snapshot := Snapshot{Models: map[string]ModelUsage{
		"five": {Remaining: 0, ResetAt: now.Add(2 * time.Hour), Exhausted: true},
	}}

	window := SummarizeWindows(snapshot, now).FiveHour
	if !window.Available || !window.Exhausted || window.Remaining != 0 {
		t.Fatalf("five-hour exhausted window = %#v", window)
	}
}
