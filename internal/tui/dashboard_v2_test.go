package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/quota"
)

func TestFriendlyDashboardEnterConfirmsHotSwitch(t *testing.T) {
	model := Model{
		Context:  context.Background(),
		Accounts: []account.Account{{ID: "work"}},
		Focus:    focusAccounts,
		Status:   "Ready",
	}
	friendly := newFriendlyDashboardModel(model)
	updated, cmd := friendly.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("Enter should open confirmation before executing")
	}
	got := updated.(friendlyDashboardModel)
	if got.ConfirmAction != actionHotReload || got.ConfirmProfile != "work" {
		t.Fatalf("confirmation = action %q profile %q", got.ConfirmAction, got.ConfirmProfile)
	}
}

func TestFriendlyAccountsPrioritizeFiveHourWindow(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	now := time.Now().UTC()
	model := Model{
		Accounts: []account.Account{{ID: "work", Active: true}},
		Results: []quota.Result{{
			Profile: "work",
			Snapshot: quota.Snapshot{
				Profile:   "work",
				Source:    "google-cloud-code",
				FetchedAt: now,
				Models: map[string]quota.ModelUsage{
					"m": {Name: "Model", Remaining: 18, ResetAt: now.Add(4 * time.Hour)},
				},
			},
		}},
		Focus: focusAccounts,
	}
	panel := stripANSI(renderFriendlyAccounts(model, 72, 10, currentTheme()))
	if !strings.Contains(panel, "5h 18%") {
		t.Fatalf("accounts panel does not prioritize five-hour quota: %q", panel)
	}
}

func TestAccountHealthDoesNotInventWeekly(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	now := time.Now().UTC()
	model := Model{
		Accounts: []account.Account{{ID: "work", Active: true}},
		Results: []quota.Result{{
			Profile: "work",
			Snapshot: quota.Snapshot{
				Profile:   "work",
				Source:    "google-cloud-code",
				FetchedAt: now,
				Models: map[string]quota.ModelUsage{
					"m": {Name: "Model", Remaining: 65, ResetAt: now.Add(3 * time.Hour)},
				},
			},
		}},
	}
	panel := stripANSI(renderAccountHealth(model, 72, 14, currentTheme()))
	if !strings.Contains(panel, "WEEKLY  not separately exposed") {
		t.Fatalf("weekly absence should be explicit: %q", panel)
	}
}
