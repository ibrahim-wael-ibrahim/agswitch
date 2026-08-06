package autoswitch

import (
	"testing"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/quota"
)

var testNow = time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

func result(profile string, remaining int) quota.Result {
	return quota.Result{
		Profile: profile,
		Snapshot: quota.Snapshot{
			Profile:   profile,
			Source:    "google-cloud-code",
			FetchedAt: testNow.Add(-time.Minute),
			Models: map[string]quota.ModelUsage{
				"model": {ID: "model", Name: "Model", Remaining: remaining},
			},
		},
	}
}

func selectTest(results []quota.Result, current string, threshold int) Decision {
	return SelectWithPolicy(results, current, threshold, Policy{Now: testNow})
}

func TestSelectSwitchesBelowThreshold(t *testing.T) {
	decision := selectTest([]quota.Result{result("work", 10), result("personal", 80)}, "work", 20)
	if !decision.Switch || decision.Selected.Profile != "personal" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestSelectKeepsCurrentAboveThreshold(t *testing.T) {
	decision := selectTest([]quota.Result{result("work", 40), result("personal", 80)}, "work", 20)
	if decision.Switch || decision.Selected.Profile != "work" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestSelectIgnoresUnknownQuota(t *testing.T) {
	decision := selectTest([]quota.Result{result("work", 10), result("unknown", -1)}, "work", 20)
	if decision.Switch || decision.Selected.Profile != "work" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestSelectRejectsStaleFallback(t *testing.T) {
	stale := result("personal", 100)
	stale.Snapshot.Source = "cache-stale"
	decision := selectTest([]quota.Result{result("work", 10), stale}, "work", 20)
	if decision.Switch || decision.Selected.Profile != "work" {
		t.Fatalf("stale quota influenced decision: %#v", decision)
	}
}

func TestSelectRejectsOldSnapshot(t *testing.T) {
	old := result("personal", 100)
	old.Snapshot.FetchedAt = testNow.Add(-10 * time.Minute)
	decision := selectTest([]quota.Result{result("work", 10), old}, "work", 20)
	if decision.Switch || decision.Selected.Profile != "work" {
		t.Fatalf("old quota influenced decision: %#v", decision)
	}
}
