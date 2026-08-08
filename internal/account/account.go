package account

import "time"

type Account struct {
	ID                    string    `json:"id"`
	Email                 string    `json:"email,omitempty"`
	Label                 string    `json:"label,omitempty"`
	Active                bool      `json:"-"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	LastUsedAt            time.Time `json:"last_used_at,omitempty"`
	CredentialFingerprint string    `json:"credential_fingerprint"`
	IdentityFingerprint   string    `json:"identity_fingerprint,omitempty"`
	QuotaEnabled          bool      `json:"quota_enabled"`
}
