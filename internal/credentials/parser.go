package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

const maxCredentialBytes = 10 << 20

func Parse(raw []byte) (Credential, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return Credential{}, errors.New("credential payload is empty")
	}
	if len(trimmed) > maxCredentialBytes {
		return Credential{}, errors.New("credential payload is too large")
	}
	if !json.Valid(trimmed) {
		return Credential{}, errors.New("credential payload is not valid JSON")
	}

	credential := New(trimmed)
	var value any
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return Credential{}, err
	}
	credential.Email = findString(value, map[string]struct{}{
		"email": {}, "user_email": {}, "account_email": {},
	})
	credential.Subject = findString(value, map[string]struct{}{
		"subject": {}, "sub": {}, "user_id": {},
	})
	credential.CredentialType = findString(value, map[string]struct{}{
		"credential_type": {}, "type": {},
	})
	return credential, nil
}

func findString(value any, keys map[string]struct{}) string {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := keys[strings.ToLower(key)]; ok {
				if text, ok := child.(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
			}
		}
		for _, child := range typed {
			if result := findString(child, keys); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range typed {
			if result := findString(child, keys); result != "" {
				return result
			}
		}
	}
	return ""
}
