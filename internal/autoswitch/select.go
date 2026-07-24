package autoswitch

import (
	"sort"

	"github.com/ibrahim-wael/agswitch/internal/quota"
)

type Candidate struct {
	Profile          string `json:"profile"`
	MinimumRemaining int    `json:"minimum_remaining"`
	Current          bool   `json:"current"`
}

type Decision struct {
	Current   Candidate   `json:"current"`
	Selected  Candidate   `json:"selected"`
	Candidates []Candidate `json:"candidates"`
	Threshold int         `json:"threshold"`
	Switch    bool        `json:"switch"`
	Reason    string      `json:"reason"`
}

// Select applies a conservative quota policy: the score for each profile is
// its lowest known model quota. Profiles with only unknown quota are ignored.
func Select(results []quota.Result, currentProfile string, threshold int) Decision {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	decision := Decision{Threshold: threshold}
	for _, result := range results {
		if result.Err != nil {
			continue
		}
		remaining, ok := quota.MinimumKnownRemaining(result.Snapshot)
		if !ok {
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
		decision.Reason = "no profiles have known quota"
		return decision
	}
	decision.Selected = decision.Candidates[0]
	if currentProfile == "" {
		decision.Switch = true
		decision.Reason = "active credential is not matched to a saved profile"
		return decision
	}
	if decision.Current.Profile == "" {
		decision.Reason = "current profile has no known quota"
		if decision.Selected.Profile != currentProfile {
			decision.Switch = true
		}
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
