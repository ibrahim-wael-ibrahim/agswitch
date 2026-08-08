package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/quota"
)

func TestQuotaAuthIssueDetectsCredentialRefreshFailures(t *testing.T) {
	result := quota.Result{
		Profile: "work",
		Snapshot: quota.Snapshot{Metadata: map[string]string{
			"warning": "access token refresh requires client_id in the credential or AGSWITCH_OAUTH_CLIENT_ID",
		}},
	}
	if !quotaAuthIssue(result) {
		t.Fatal("expected OAuth configuration warning to be classified as auth")
	}
}

func TestRetryableAccountsSkipsLiveAndAuthFailures(t *testing.T) {
	now := time.Now().UTC()
	accounts := []account.Account{{ID: "live"}, {ID: "auth"}, {ID: "retry"}}
	results := []quota.Result{
		{
			Profile: "live",
			Snapshot: quota.Snapshot{
				Source:    "google-cloud-code",
				FetchedAt: now,
				Models: map[string]quota.ModelUsage{
					"m": {ID: "m", Name: "m", Remaining: 80},
				},
			},
		},
		{
			Profile: "auth",
			Snapshot: quota.Snapshot{
				Source:   "cache-stale",
				Metadata: map[string]string{"warning": "client_secret is missing"},
			},
		},
		{Profile: "retry", Err: errors.New("quota API returned HTTP 503")},
	}

	pending := retryableAccounts(accounts, results, now)
	if len(pending) != 1 || pending[0].ID != "retry" {
		t.Fatalf("pending = %#v; want only retry", pending)
	}
}

func TestDashboardQuotaStatusDistinguishesAuthFromTransient(t *testing.T) {
	now := time.Now().UTC()
	_, state := dashboardQuotaStatus(quota.Result{
		Snapshot: quota.Snapshot{Metadata: map[string]string{"warning": "client_id missing"}},
	}, now)
	if state != "AUTH" {
		t.Fatalf("auth state = %q; want AUTH", state)
	}

	_, state = dashboardQuotaStatus(quota.Result{Err: errors.New("quota API returned HTTP 503")}, now)
	if state != "RETRY" {
		t.Fatalf("transient state = %q; want RETRY", state)
	}
}

func TestMergeQuotaResultsReplacesProfilesInPlace(t *testing.T) {
	base := []quota.Result{{Profile: "a"}, {Profile: "b"}}
	updated := mergeQuotaResults(base, []quota.Result{{Profile: "b", Err: errors.New("new")}, {Profile: "c"}})
	if len(updated) != 3 {
		t.Fatalf("len = %d; want 3", len(updated))
	}
	if updated[1].Err == nil || updated[1].Err.Error() != "new" {
		t.Fatalf("profile b was not replaced: %#v", updated[1])
	}
	if updated[2].Profile != "c" {
		t.Fatalf("last profile = %q; want c", updated[2].Profile)
	}
}
