package quota

import "time"

type ModelUsage struct {
	Name      string    `json:"name"`
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	ResetAt   time.Time `json:"reset_at"`
}

type Snapshot struct {
	Email            string                `json:"email"`
	SubscriptionTier string                `json:"subscription_tier"`
	Models           map[string]ModelUsage `json:"models"`
	FetchedAt        time.Time             `json:"fetched_at"`
	Source           string                `json:"source,omitempty"`
	Metadata         map[string]string     `json:"metadata,omitempty"`
}
