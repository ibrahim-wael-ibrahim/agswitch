package quota

import (
	"strings"
	"time"
)

const DefaultAutoSwitchMaxAge = 2 * time.Minute

// EligibleForAutoSwitch rejects fallback and stale data. A fresh local cache is
// allowed only when it originated from the live Google Cloud Code provider.
func EligibleForAutoSwitch(snapshot Snapshot, now time.Time, maxAge time.Duration) (bool, string) {
	if strings.TrimSpace(snapshot.Source) != "google-cloud-code" {
		return false, "quota source is not live Google Cloud Code"
	}
	if warning := strings.TrimSpace(snapshot.Metadata["warning"]); warning != "" {
		return false, "quota snapshot contains a provider warning"
	}
	if snapshot.FetchedAt.IsZero() {
		return false, "quota snapshot has no fetch time"
	}
	if maxAge <= 0 {
		maxAge = DefaultAutoSwitchMaxAge
	}
	age := now.Sub(snapshot.FetchedAt)
	if age < 0 {
		age = 0
	}
	if age > maxAge {
		return false, "quota snapshot is too old"
	}
	return true, ""
}
