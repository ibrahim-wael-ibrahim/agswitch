package quota

import (
	"sort"
	"strings"
	"time"
)

// SortedModels returns a display-oriented view. Internal model IDs that share
// the same display name are grouped, using the lowest known remaining quota so
// the dashboard never overstates availability.
func SortedModels(snapshot Snapshot) []ModelUsage {
	grouped := make(map[string]ModelUsage, len(snapshot.Models))
	for _, model := range snapshot.Models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = model.ID
		}
		key := strings.ToLower(name)
		current, exists := grouped[key]
		if !exists {
			model.Name = name
			model.Variants = 1
			grouped[key] = model
			continue
		}
		current.Variants++
		if current.ID == "" {
			current.ID = model.ID
		}
		if model.Remaining >= 0 && (current.Remaining < 0 || model.Remaining < current.Remaining) {
			current.Remaining = model.Remaining
			current.Limit = model.Limit
			current.Exhausted = model.Exhausted
		}
		if !model.ResetAt.IsZero() && (current.ResetAt.IsZero() || model.ResetAt.Before(current.ResetAt)) {
			current.ResetAt = model.ResetAt
		}
		grouped[key] = current
	}

	models := make([]ModelUsage, 0, len(grouped))
	for _, model := range grouped {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		leftKnown := models[i].Remaining >= 0
		rightKnown := models[j].Remaining >= 0
		if leftKnown != rightKnown {
			return leftKnown
		}
		if models[i].Remaining != models[j].Remaining {
			return models[i].Remaining < models[j].Remaining
		}
		return models[i].Name < models[j].Name
	})
	return models
}

// MinimumKnownRemaining returns the lowest known quota percentage for a
// snapshot. Unknown-only snapshots return ok=false.
func MinimumKnownRemaining(snapshot Snapshot) (remaining int, ok bool) {
	for _, model := range SortedModels(snapshot) {
		if model.Remaining < 0 {
			continue
		}
		if !ok || model.Remaining < remaining {
			remaining = model.Remaining
			ok = true
		}
	}
	return remaining, ok
}

func ResetIn(resetAt, now time.Time) time.Duration {
	if resetAt.IsZero() {
		return 0
	}
	value := resetAt.Sub(now)
	if value < 0 {
		return 0
	}
	return value.Round(time.Minute)
}
