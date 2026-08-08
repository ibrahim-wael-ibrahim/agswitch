package credentials

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func Fingerprint(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// IdentityFingerprint returns a stable, non-secret account identifier. It
// prefers immutable Google identity claims, then falls back to the refresh
// token because Antigravity credentials observed in the wild often omit email
// and subject fields. The original identity value is never stored.
func IdentityFingerprint(subject, email, refreshToken string) string {
	var material string
	switch {
	case strings.TrimSpace(subject) != "":
		material = "subject:" + strings.TrimSpace(subject)
	case strings.TrimSpace(email) != "":
		material = "email:" + strings.ToLower(strings.TrimSpace(email))
	case strings.TrimSpace(refreshToken) != "":
		material = "refresh-token:" + strings.TrimSpace(refreshToken)
	default:
		return ""
	}
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}
