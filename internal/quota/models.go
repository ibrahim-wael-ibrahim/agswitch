package quota

import "time"

type ModelUsage struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	ResetAt   time.Time `json:"reset_at,omitempty"`
	Exhausted bool      `json:"exhausted,omitempty"`
}

type Snapshot struct {
	Profile          string                `json:"profile"`
	Email            string                `json:"email"`
	SubscriptionTier string                `json:"subscription_tier,omitempty"`
	Models           map[string]ModelUsage `json:"models"`
	FetchedAt        time.Time             `json:"fetched_at"`
	Source           string                `json:"source,omitempty"`
	Cached           bool                  `json:"-"`
	Metadata         map[string]string     `json:"metadata,omitempty"`
}

type Result struct {
	Profile  string
	Snapshot Snapshot
	Err      error
}
