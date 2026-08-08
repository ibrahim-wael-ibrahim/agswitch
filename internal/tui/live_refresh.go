package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/autoswitch"
	"github.com/ibrahim-wael/agswitch/internal/quota"
)

const liveRefreshAttempts = 4

var liveRefreshBackoff = []time.Duration{
	750 * time.Millisecond,
	1500 * time.Millisecond,
	3 * time.Second,
}

type liveQuotaMsg struct {
	data      dataMsg
	live      int
	total     int
	auth      int
	transient int
}

func (m friendlyDashboardModel) Init() tea.Cmd {
	return m.Model.loadLiveQuotaCommand()
}

func (m Model) loadLiveQuotaCommand() tea.Cmd {
	return func() tea.Msg {
		accounts, err := m.Backend.List(m.Context)
		if err != nil {
			return liveQuotaMsg{data: dataMsg{err: err, refresh: true}}
		}

		results := m.Backend.FetchAll(m.Context, accounts, true)
		for attempt := 1; attempt < liveRefreshAttempts; attempt++ {
			pending := retryableAccounts(accounts, results, time.Now().UTC())
			if len(pending) == 0 {
				break
			}
			if err := waitForRetry(m.Context, liveRefreshBackoff[attempt-1]); err != nil {
				break
			}
			retried := m.Backend.FetchAll(m.Context, pending, true)
			results = mergeQuotaResults(results, retried)
		}

		current := ""
		for _, item := range accounts {
			if item.Active {
				current = item.ID
				break
			}
		}

		message := liveQuotaMsg{
			data: dataMsg{
				accounts: accounts,
				results:  results,
				decision: autoswitch.Select(results, current, m.Options.AutoThreshold),
				refresh:  true,
			},
			total: len(accounts),
		}
		for _, result := range results {
			if ok, _ := quota.EligibleForAutoSwitch(result.Snapshot, time.Now().UTC(), quota.DefaultAutoSwitchMaxAge); result.Err == nil && ok {
				message.live++
				continue
			}
			if quotaAuthIssue(result) {
				message.auth++
			} else {
				message.transient++
			}
		}
		return message
	}
}

func (m liveQuotaMsg) statusText() string {
	if m.data.err != nil {
		return "Live quota refresh failed: " + m.data.err.Error()
	}
	status := fmt.Sprintf("Live quota %d/%d accounts", m.live, m.total)
	parts := make([]string, 0, 2)
	if m.auth > 0 {
		parts = append(parts, fmt.Sprintf("%d need auth", m.auth))
	}
	if m.transient > 0 {
		parts = append(parts, fmt.Sprintf("%d will retry", m.transient))
	}
	if len(parts) > 0 {
		status += " · " + strings.Join(parts, " · ")
	} else if m.total > 0 {
		status += " · all current"
	}
	return status
}

func retryableAccounts(accounts []account.Account, results []quota.Result, now time.Time) []account.Account {
	byProfile := make(map[string]quota.Result, len(results))
	for _, result := range results {
		byProfile[result.Profile] = result
	}
	pending := make([]account.Account, 0, len(accounts))
	for _, item := range accounts {
		result, ok := byProfile[item.ID]
		if !ok {
			pending = append(pending, item)
			continue
		}
		if eligible, _ := quota.EligibleForAutoSwitch(result.Snapshot, now, quota.DefaultAutoSwitchMaxAge); result.Err == nil && eligible {
			continue
		}
		if quotaAuthIssue(result) {
			continue
		}
		pending = append(pending, item)
	}
	return pending
}

func mergeQuotaResults(base, updates []quota.Result) []quota.Result {
	index := make(map[string]int, len(base))
	for i, result := range base {
		index[result.Profile] = i
	}
	for _, result := range updates {
		if i, ok := index[result.Profile]; ok {
			base[i] = result
			continue
		}
		index[result.Profile] = len(base)
		base = append(base, result)
	}
	return base
}

func quotaAuthIssue(result quota.Result) bool {
	text := ""
	if result.Err != nil {
		text = result.Err.Error()
	}
	if warning := strings.TrimSpace(result.Snapshot.Metadata["warning"]); warning != "" {
		text += " " + warning
	}
	text = strings.ToLower(text)
	for _, marker := range []string{
		"client_id",
		"client secret",
		"client_secret",
		"no refresh token",
		"unauthenticated",
		"invalid authentication",
		"http 401",
		"http 403",
		"forbidden",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func dashboardQuotaStatus(result quota.Result, now time.Time) (bool, string) {
	if quotaAuthIssue(result) {
		return false, "AUTH"
	}
	if result.Err != nil {
		return false, "RETRY"
	}
	snapshot := result.Snapshot
	if warning := strings.TrimSpace(snapshot.Metadata["warning"]); warning != "" || snapshot.Source == "cache-stale" {
		return false, "STALE"
	}
	if snapshot.Source != "google-cloud-code" || snapshot.FetchedAt.IsZero() {
		return false, "UNKNOWN"
	}
	if now.Sub(snapshot.FetchedAt) > quota.DefaultAutoSwitchMaxAge {
		return false, "OLD"
	}
	if _, ok := quota.MinimumKnownRemaining(snapshot); !ok {
		return false, "UNKNOWN"
	}
	return true, "LIVE"
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
