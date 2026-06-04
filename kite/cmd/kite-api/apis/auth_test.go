package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
	"kite/internal/config"
)

func TestLoginIssuesAccessToken(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(`{"email":"admin@example.com","password":"admin"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.AccessToken == "" {
		t.Fatal("expected accessToken")
	}

	if res.ExpiresIn != int64(time.Hour.Seconds()) {
		t.Fatalf("expected 60 minute expiry, got %d seconds", res.ExpiresIn)
	}

	if res.TokenType != "Bearer" {
		t.Fatalf("expected Bearer token type, got %q", res.TokenType)
	}

	claims, err := newTestTokenService(t).VerifyAccessToken(res.AccessToken)
	if err != nil {
		t.Fatalf("failed to verify access token: %v", err)
	}

	if claims.AccessLevel != auth.AccessLevelAdmin {
		t.Fatalf("expected admin access level, got %d", claims.AccessLevel)
	}

	cookie := rec.Result().Cookies()[0]
	if cookie.Name != "accessToken" {
		t.Fatalf("expected accessToken cookie, got %q", cookie.Name)
	}

	if !strings.HasPrefix(cookie.Value, "Bearer ") {
		t.Fatalf("expected Bearer cookie value, got %q", cookie.Value)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(`{"email":"admin@example.com","password":"wrong"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		JWTSecret:      "test-secret",
		PasswordSalt:   "test-salt",
		AccessTokenTTL: time.Hour,
	}

	tokenService, err := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	r := gin.Default()
	api := r.Group("/api")
	Register(api, Dependencies{
		Config:       cfg,
		TokenService: tokenService,
		DynamicClient: fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			userTestGVR: "KiteUserList",
		}, newLoginTestUser("admin", auth.HashPassword("admin", cfg.PasswordSalt), auth.AccessLevelAdmin)),
	})

	return r
}

var userTestGVR = schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kiteusers",
}

func newLoginTestUser(username string, passwordHash string, accessLevel int) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "anacnu.com/v1",
			"kind":       "KiteUser",
			"metadata": map[string]any{
				"name": username,
			},
			"spec": map[string]any{
				"username":      username,
				"email":         username + "@example.com",
				"password":      passwordHash,
				"namespace":     username,
				"profile_image": "",
				"access_level":  int64(accessLevel),
			},
		},
	}
}
