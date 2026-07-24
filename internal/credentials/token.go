package credentials

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// TokenStatus contains only non-secret facts about an OAuth access token.
type TokenStatus struct {
	AccessTokenPresent bool
	Expiry             time.Time
	ExpiryKnown        bool
}

func (s TokenStatus) Fresh(now time.Time, skew time.Duration) bool {
	if !s.AccessTokenPresent {
		return false
	}
	if !s.ExpiryKnown {
		return true
	}
	return s.Expiry.After(now.Add(skew))
}

// InspectToken extracts token presence and expiry without returning token values.
func InspectToken(raw []byte) TokenStatus {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return TokenStatus{}
	}
	status := TokenStatus{
		AccessTokenPresent: findNonEmptyString(value, "accesstoken"),
	}
	if expiry, ok := findExpiry(value); ok {
		status.Expiry = expiry
		status.ExpiryKnown = true
	}
	return status
}

func findNonEmptyString(value any, wanted string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if normalizeTokenKey(key) == wanted {
				text, ok := child.(string)
				return ok && strings.TrimSpace(text) != ""
			}
		}
		for _, child := range typed {
			if findNonEmptyString(child, wanted) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if findNonEmptyString(child, wanted) {
				return true
			}
		}
	}
	return false
}

func findExpiry(value any) (time.Time, bool) {
	wanted := map[string]struct{}{
		"expiry": {}, "expiresat": {}, "expiration": {}, "expirationtime": {},
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := wanted[normalizeTokenKey(key)]; ok {
				if expiry, parsed := parseExpiry(child); parsed {
					return expiry, true
				}
			}
		}
		for _, child := range typed {
			if expiry, ok := findExpiry(child); ok {
				return expiry, true
			}
		}
	case []any:
		for _, child := range typed {
			if expiry, ok := findExpiry(child); ok {
				return expiry, true
			}
		}
	}
	return time.Time{}, false
}

func parseExpiry(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return time.Time{}, false
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if parsed, err := time.Parse(layout, text); err == nil {
				return parsed.UTC(), true
			}
		}
		if numeric, err := strconv.ParseInt(text, 10, 64); err == nil {
			return unixExpiry(numeric), true
		}
	case float64:
		return unixExpiry(int64(typed)), true
	case json.Number:
		if numeric, err := typed.Int64(); err == nil {
			return unixExpiry(numeric), true
		}
	}
	return time.Time{}, false
}

func unixExpiry(value int64) time.Time {
	// Current Unix milliseconds are 13 digits; seconds are 10 digits.
	if value > 100_000_000_000 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}

func normalizeTokenKey(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}
