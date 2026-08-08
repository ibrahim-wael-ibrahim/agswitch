package quota

import "time"

const (
	// FiveHourWindowMax keeps the UI grounded in the provider's actual resetTime
	// while allowing a little clock/network skew around Antigravity's documented
	// five-hour refresh window.
	FiveHourWindowMax = 6 * time.Hour
	WeeklyWindowMax   = 8 * 24 * time.Hour
)

type WindowSummary struct {
	Available bool      `json:"available"`
	Remaining int       `json:"remaining,omitempty"`
	ResetAt   time.Time `json:"reset_at,omitempty"`
	Exhausted bool      `json:"exhausted,omitempty"`
	Models    int       `json:"models,omitempty"`
}

type WindowSet struct {
	FiveHour WindowSummary `json:"five_hour"`
	Weekly   WindowSummary `json:"weekly"`
}

// SummarizeWindows groups model quota by the reset time returned by the
// provider. A reset within six hours is shown as the five-hour window. A reset
// farther out (up to eight days) is shown as a weekly/long window.
//
// Google currently returns one quotaInfo object per model, not two independent
// counters for five-hour and weekly quota. Consequently Weekly.Available is
// false unless the provider actually returns a long reset window. The caller
// must not invent a weekly percentage when it is absent.
func SummarizeWindows(snapshot Snapshot, now time.Time) WindowSet {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	var output WindowSet
	for _, model := range snapshot.Models {
		if model.Remaining < 0 || model.ResetAt.IsZero() {
			continue
		}
		resetAt := model.ResetAt.UTC()
		delta := resetAt.Sub(now)
		if delta <= 0 || delta > WeeklyWindowMax {
			continue
		}
		if delta <= FiveHourWindowMax {
			mergeWindow(&output.FiveHour, model, resetAt)
			continue
		}
		mergeWindow(&output.Weekly, model, resetAt)
	}
	return output
}

func mergeWindow(window *WindowSummary, model ModelUsage, resetAt time.Time) {
	window.Models++
	if !window.Available {
		window.Available = true
		window.Remaining = model.Remaining
		window.ResetAt = resetAt
		window.Exhausted = model.Exhausted || model.Remaining == 0
		return
	}

	// The dashboard is intended to answer "which account is constrained now?".
	// Use the most constrained model in each provider reset window.
	if model.Remaining < window.Remaining {
		window.Remaining = model.Remaining
		window.ResetAt = resetAt
	} else if model.Remaining == window.Remaining && resetAt.Before(window.ResetAt) {
		window.ResetAt = resetAt
	}
	window.Exhausted = window.Exhausted || model.Exhausted || model.Remaining == 0
}
