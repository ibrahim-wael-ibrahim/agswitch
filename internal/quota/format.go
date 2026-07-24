package quota

import (
	"sort"
	"time"
)

func SortedModels(snapshot Snapshot) []ModelUsage {
	models := make([]ModelUsage, 0, len(snapshot.Models))
	for _, model := range snapshot.Models {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Remaining != models[j].Remaining {
			return models[i].Remaining < models[j].Remaining
		}
		return models[i].Name < models[j].Name
	})
	return models
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
