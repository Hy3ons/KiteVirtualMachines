package auth

import (
	"testing"
	"time"
)

func TestTokenLifetimeCannotExceedConfiguredTTL(t *testing.T) {
	service, err := NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, lifetime := range []time.Duration{10 * time.Minute, 2 * time.Hour} {
		token, _, err := service.IssueAccessTokenWithLifetime("member", AccessLevelUser, lifetime)
		if err != nil {
			t.Fatal(err)
		}
		claims, err := service.VerifyAccessToken(token)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := claims.ExpiresAt-claims.IssuedAt, int64(min(lifetime, time.Hour).Seconds()); got != want {
			t.Fatalf("expected token lifetime %d seconds, got %d", want, got)
		}
	}
	if _, _, err := service.IssueAccessTokenWithLifetime("member", AccessLevelUser, 0); err == nil {
		t.Fatal("expected zero lifetime to be rejected")
	}
}
