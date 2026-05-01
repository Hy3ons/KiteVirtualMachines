package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kite/internal/auth"
	"kite/internal/config"
)

func TestHealth(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:       ":8080",
		JWTSecret:      "test-secret",
		AdminUsername:  "admin",
		AdminPassword:  "admin",
		AdminAccess:    auth.AccessLevelAdmin,
		AccessTokenTTL: time.Hour,
	}

	tokenService, err := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	r := newRouter(cfg, tokenService, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
