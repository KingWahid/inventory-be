package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Hex returns lowercase hex-encoded SHA-256 of raw bytes (for movement POST body fingerprinting).
func SHA256Hex(raw []byte) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
