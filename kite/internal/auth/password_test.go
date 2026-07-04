package auth

import "testing"

func TestHashPasswordUsesSaltedBcrypt(t *testing.T) {
	firstHash, err := HashPassword("secret-password", "legacy-salt")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	secondHash, err := HashPassword("secret-password", "legacy-salt")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if firstHash == secondHash {
		t.Fatal("expected bcrypt hashes for the same password to differ")
	}
	if !VerifyPassword("secret-password", "legacy-salt", firstHash) {
		t.Fatal("expected bcrypt hash to verify")
	}
	if PasswordNeedsRehash(firstHash) {
		t.Fatal("expected bcrypt hash to be current")
	}
}

func TestVerifyPasswordAcceptsLegacyHashForMigration(t *testing.T) {
	legacyHash := LegacyHashPassword("secret-password", "legacy-salt")

	if !VerifyPassword("secret-password", "legacy-salt", legacyHash) {
		t.Fatal("expected legacy hash to verify")
	}
	if !PasswordNeedsRehash(legacyHash) {
		t.Fatal("expected legacy hash to need rehash")
	}
}
