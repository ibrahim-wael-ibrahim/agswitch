package quota

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode"
)

type AuthMaterial struct {
	AccessToken  string
	RefreshToken string
	ClientID     string
	ClientSecret string
	ProjectID    string
}

func ExtractAuth(raw []byte) (AuthMaterial, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return AuthMaterial{}, err
	}
	auth := AuthMaterial{
		AccessToken:  findAuthString(value, "accesstoken"),
		RefreshToken: findAuthString(value, "refreshtoken"),
		ClientID:     findAuthString(value, "clientid", "oauthclientid"),
		ClientSecret: findAuthString(value, "clientsecret", "oauthclientsecret"),
		ProjectID:    findAuthString(value, "projectid", "managedprojectid", "cloudaicompanionproject"),
	}
	if index := strings.LastIndex(auth.RefreshToken, "|"); index > 0 {
		if auth.ProjectID == "" && index+1 < len(auth.RefreshToken) {
			auth.ProjectID = strings.TrimSpace(auth.RefreshToken[index+1:])
		}
		auth.RefreshToken = strings.TrimSpace(auth.RefreshToken[:index])
	}
	if auth.AccessToken == "" && auth.RefreshToken == "" {
		return AuthMaterial{}, errors.New("credential does not contain an access token or refresh token")
	}
	return auth, nil
}

func findAuthString(value any, names ...string) string {
	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[normalizeKey(name)] = struct{}{}
	}
	return findAuthStringValue(value, wanted)
}

func findAuthStringValue(value any, wanted map[string]struct{}) string {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := wanted[normalizeKey(key)]; ok {
				switch v := child.(type) {
				case string:
					if text := strings.TrimSpace(v); text != "" {
						return text
					}
				case map[string]any:
					if text := findAuthStringValue(v, map[string]struct{}{"id": {}}); text != "" {
						return text
					}
				}
			}
		}
		for _, child := range typed {
			if text := findAuthStringValue(child, wanted); text != "" {
				return text
			}
		}
	case []any:
		for _, child := range typed {
			if text := findAuthStringValue(child, wanted); text != "" {
				return text
			}
		}
	}
	return ""
}

func normalizeKey(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}
