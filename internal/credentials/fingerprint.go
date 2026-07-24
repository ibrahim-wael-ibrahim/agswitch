package credentials

import (
	"crypto/sha256"
	"encoding/hex"
)

func Fingerprint(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
