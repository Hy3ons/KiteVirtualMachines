package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

const passwordHashIterations = 100_000

// HashPassword creates an iterated SHA-256 password hash with a shared salt.
// password is the plain text password received from a user or VM credential request.
// salt is the server-side value loaded from runtime configuration.
// The returned string is a hex-encoded hash expected to be stored in KiteUser spec.password or KiteVirtualMachine spec.sshPasswordHash.
// This function is used by API authentication, user creation, and gateway SSH authentication code.
func HashPassword(password string, salt string) string {
	sum := sha256.Sum256([]byte(salt + password))
	for range passwordHashIterations - 1 {
		sum = sha256.Sum256(sum[:])
	}

	return hex.EncodeToString(sum[:])
}

// VerifyPassword checks a plain text password against a stored password hash.
// password is the plain text password received from an API login or SSH gateway request.
// salt is the server-side value loaded from runtime configuration.
// expectedHash is the hex-encoded value stored in KiteUser spec.password or KiteVirtualMachine spec.sshPasswordHash.
// The returned value is true only when the computed hash matches in constant time.
func VerifyPassword(password string, salt string, expectedHash string) bool {
	return constantTimeEqual(HashPassword(password, salt), expectedHash)
}
