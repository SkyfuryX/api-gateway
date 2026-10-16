package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashAPIKey returns the SHA-256 string representation of a raw API key
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
