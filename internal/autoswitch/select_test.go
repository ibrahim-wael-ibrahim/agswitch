package autoswitch

import (
	"testing"

	"github.com/ibrahim-wael/agswitch/internal/quota"
)

func result(profile string, remaining int) quota.Result {
	return quota.Result{
		Profile: profile,
		Snapshot: quota.Snapshot{
			Profile: profile,
			Models: map[string]quota.ModelUsage{
				"model": {ID: "model", Name: "Model", Remaining: remaining},
			},
		},
	}
}

func TestSelectSwitchesBelowThreshold(t *testing.T) {
	decision := Select([]quota.Result{result("work", 10), result("personal", 80)}, "work", 20)
	if !decision.Switch || decision.Selected.Profile != "personal" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestSelectKeepsCurrentAboveThreshold(t *testing.T) {
	decision := Select([]quota.Result{result("work", 40), result("personal", 80)}, "work", 20)
	if decision.Switch || decision.Selected.Profile != "work" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestSelectIgnoresUnknownQuota(t *testing.T) {
	decision := Select([]quota.Result{result("work", 10), result("unknown", -1)}, "work", 20)
	if decision.Switch || decision.Selected.Profile != "work" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}
