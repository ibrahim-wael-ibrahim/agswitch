package credentials

import "encoding/json"

type Credential struct {
	Raw                 []byte `json:"raw"`
	Email               string `json:"email,omitempty"`
	Subject             string `json:"subject,omitempty"`
	Fingerprint         string `json:"fingerprint,omitempty"`
	Source              string `json:"source,omitempty"`
	CredentialType      string `json:"credential_type,omitempty"`
}

func New(raw []byte) Credential {
	clone := append([]byte(nil), raw...)
	return Credential{
		Raw:         clone,
		Fingerprint: Fingerprint(clone),
	}
}

func (c Credential) MarshalJSON() ([]byte, error) {
	type alias Credential
	if len(c.Raw) == 0 {
		c.Raw = nil
	}
	return json.Marshal(alias(c))
}
