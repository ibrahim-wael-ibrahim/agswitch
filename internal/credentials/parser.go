package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
)

type parsedCredential struct {
	Email   string `json:"email"`
	Subject string `json:"subject"`
}

func Parse(raw []byte) (Credential, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return Credential{}, errors.New("credential payload is empty")
	}

	if !json.Valid(trimmed) {
		return Credential{}, errors.New("credential payload is not valid JSON")
	}

	credential := New(trimmed)

	var parsed parsedCredential
	if err := json.Unmarshal(trimmed, &parsed); err == nil {
		credential.Email = parsed.Email
		credential.Subject = parsed.Subject
	}

	return credential, nil
}
