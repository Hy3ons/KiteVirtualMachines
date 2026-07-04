package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	passwordHashIterations = 100_000
	bcryptCost             = 12
)

// HashPassword creates a bcrypt password hash.
// password is the plain text password received from a user or VM credential request.
// salt is accepted for backward-compatible callers but is not used by bcrypt.
// The returned string includes bcrypt salt and cost metadata for future verification.
// This function is used by API authentication, user creation, and gateway SSH authentication code.
func HashPassword(password string, salt string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func legacyHashPassword(password string, salt string) string {
	sum := sha256.Sum256([]byte(salt + password))
	for range passwordHashIterations - 1 {
		sum = sha256.Sum256(sum[:])
	}

	return hex.EncodeToString(sum[:])
}

// VerifyPassword checks a plain text password against a stored password hash.
// password is the plain text password received from an API login or SSH gateway request.
// salt is the server-side value loaded from runtime configuration.
// expectedHash is the bcrypt or legacy SHA-256 value stored in KiteUser spec.password or KiteVirtualMachine spec.sshPasswordHash.
// The returned value is true only when the stored hash verifies.
func VerifyPassword(password string, salt string, expectedHash string) bool {
	if isBcryptHash(expectedHash) {
		return bcrypt.CompareHashAndPassword([]byte(expectedHash), []byte(password)) == nil
	}

	return constantTimeEqual(legacyHashPassword(password, salt), expectedHash)
}

// PasswordNeedsRehash reports whether a verified password should be rewritten with the current bcrypt format.
// expectedHash is the stored KiteUser or KiteVirtualMachine password hash.
// This function is used by login migration code after VerifyPassword succeeds.
func PasswordNeedsRehash(expectedHash string) bool {
	return !isBcryptHash(expectedHash)
}

// LegacyHashPassword creates the retired shared-salt SHA-256 hash for compatibility tests and migration reads.
// password is the plain text password to hash.
// salt is the historical runtime password salt.
// The returned hash is accepted only by VerifyPassword compatibility logic.
func LegacyHashPassword(password string, salt string) string {
	return legacyHashPassword(password, salt)
}

func isBcryptHash(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") ||
		strings.HasPrefix(hash, "$2b$") ||
		strings.HasPrefix(hash, "$2y$")
}
