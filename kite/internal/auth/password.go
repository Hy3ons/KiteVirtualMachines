package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

const passwordHashIterations = 100_000

// HashPassword creates an iterated SHA-256 password hash with a shared salt.
// password is the plain text password received from a login or user creation request.
// salt is the server-side value loaded from environment configuration.
// The returned string is a hex-encoded hash expected to be stored in KiteUser spec.password.
// This function is used by API authentication and user creation code.
func HashPassword(password string, salt string) string {
	sum := sha256.Sum256([]byte(salt + password))
	for range passwordHashIterations - 1 {
		sum = sha256.Sum256(sum[:])
	}

	return hex.EncodeToString(sum[:])
}
