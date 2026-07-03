package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
	"kite/internal/config"
	"kite/internal/health"
)

func TestHealth(t *testing.T) {
	r := newHealthTestRouter(t, fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		healthTestKiteUserGVR:           "KiteUserList",
		healthTestKiteVirtualMachineGVR: "KiteVirtualMachineList",
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var report health.Report
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("failed to decode health report: %v", err)
	}
	if report.Status != "ok" {
		t.Fatalf("expected ok health report, got %+v", report)
	}
}

func TestHealthFailsWithoutDynamicClient(t *testing.T) {
	r := newHealthTestRouter(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d: %s", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}
}

func newHealthTestRouter(t *testing.T, dynamicClient dynamic.Interface) http.Handler {
	t.Helper()

	cfg := config.Config{
		JWTSecret:      "test-secret",
		PasswordSalt:   "test-salt",
		AccessTokenTTL: time.Hour,
	}

	tokenService, err := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	return newRouter(cfg, tokenService, dynamicClient, nil)
}

var healthTestKiteUserGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kiteusers",
}

var healthTestKiteVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}
