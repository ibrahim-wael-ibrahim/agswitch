package autoswitch

import (
	"sort"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/quota"
)

type Candidate struct {
	Profile          string `json:"profile"`
	MinimumRemaining int    `json:"minimum_remaining"`
	Current          bool   `json:"current"`
}

type Decision struct {
	Current    Candidate   `json:"current"`
	Selected   Candidate   `json:"selected"`
	Candidates []Candidate `json:"candidates"`
	Threshold  int         `json:"threshold"`
	Switch     bool        `json:"switch"`
	Reason     string      `json:"reason"`
	Excluded   int         `json:"excluded"`
}

type Policy struct {
	Now    time.Time
	MaxAge time.Duration
}

// Select applies the default safe policy. Only recent live quota snapshots are
// eligible; stale-cache fallback is never used to change accounts.
func Select(results []quota.Result, currentProfile string, threshold int) Decision {
	return SelectWithPolicy(results, currentProfile, threshold, Policy{})
}

func SelectWithPolicy(results []quota.Result, currentProfile string, threshold int, policy Policy) Decision {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	decision := Decision{Threshold: threshold}
	for _, result := range results {
		if result.Err != nil {
			decision.Excluded++
			continue
		}
		if ok, _ := quota.EligibleForAutoSwitch(result.Snapshot, now, policy.MaxAge); !ok {
			decision.Excluded++
			continue
		}
		remaining, ok := quota.MinimumKnownRemaining(result.Snapshot)
		if !ok {
			decision.Excluded++
			continue
		}
		candidate := Candidate{
			Profile:          result.Profile,
			MinimumRemaining: remaining,
			Current:          result.Profile == currentProfile,
		}
		decision.Candidates = append(decision.Candidates, candidate)
		if candidate.Current {
			decision.Current = candidate
		}
	}
	sort.Slice(decision.Candidates, func(i, j int) bool {
		if decision.Candidates[i].MinimumRemaining != decision.Candidates[j].MinimumRemaining {
			return decision.Candidates[i].MinimumRemaining > decision.Candidates[j].MinimumRemaining
		}
		return decision.Candidates[i].Profile < decision.Candidates[j].Profile
	})
	if len(decision.Candidates) == 0 {
		decision.Reason = "no profiles have recent live quota"
		return decision
	}
	decision.Selected = decision.Candidates[0]
	if currentProfile == "" {
		decision.Reason = "active credential is not matched to a saved profile"
		return decision
	}
	if decision.Current.Profile == "" {
		decision.Reason = "current profile has no recent live quota"
		return decision
	}
	if decision.Current.MinimumRemaining > threshold {
		decision.Selected = decision.Current
		decision.Reason = "current profile is above the threshold"
		return decision
	}
	if decision.Selected.Profile == currentProfile {
		decision.Reason = "current profile is already the best available option"
		return decision
	}
	if decision.Selected.MinimumRemaining <= decision.Current.MinimumRemaining {
		decision.Selected = decision.Current
		decision.Reason = "no profile has more known quota than the current profile"
		return decision
	}
	decision.Switch = true
	decision.Reason = "current profile reached the quota threshold"
	return decision
}
