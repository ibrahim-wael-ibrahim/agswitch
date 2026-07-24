package account

import "time"

type Account struct {
	ID                   string
	Email                string
	Label                string
	Active               bool
	CreatedAt            time.Time
	LastUsedAt           time.Time
	CredentialFingerprint string
}
