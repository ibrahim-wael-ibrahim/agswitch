package credentials

// Credential keeps the original Antigravity JSON payload untouched. Parsed
// fields are metadata only and are never used to reconstruct the secret.
type Credential struct {
	Raw            []byte `json:"-"`
	Email          string `json:"email,omitempty"`
	Subject        string `json:"subject,omitempty"`
	Fingerprint    string `json:"fingerprint,omitempty"`
	Source         string `json:"source,omitempty"`
	CredentialType string `json:"credential_type,omitempty"`
}

func New(raw []byte) Credential {
	clone := append([]byte(nil), raw...)
	return Credential{Raw: clone, Fingerprint: Fingerprint(clone)}
}
