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
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.ExpiresIn != int64(time.Hour.Seconds()) {
		t.Fatalf("expected 60 minute expiry, got %d seconds", res.ExpiresIn)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to decode raw response: %v", err)
	}
	if _, ok := raw["accessToken"]; ok {
		t.Fatal("login response must not expose accessToken")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "accessToken" {
		t.Fatalf("expected accessToken cookie, got %q", cookie.Name)
	}
	if !strings.HasPrefix(cookie.Value, "Bearer ") {
		t.Fatalf("expected Bearer cookie value, got %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected accessToken cookie to be HTTP-only")
	}
	if !cookie.Secure {
		t.Fatal("expected accessToken cookie to require Secure transport")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}

	claims, err := newTestTokenService(t).VerifyAccessToken(cookie.Value)
	if err != nil {
		t.Fatalf("failed to verify access token cookie: %v", err)
	}
	if claims.AccessLevel != auth.AccessLevelAdmin {
		t.Fatalf("expected admin access level, got %d", claims.AccessLevel)
	}
}

func TestLogoutClearsAccessTokenCookie(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Result().Header.Get("Set-Cookie"), "accessToken=;") {
		t.Fatalf("expected logout to clear accessToken cookie, got %q", rec.Result().Header.Get("Set-Cookie"))
	}
	if !strings.Contains(rec.Result().Header.Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("expected logout cookie Max-Age=0, got %q", rec.Result().Header.Get("Set-Cookie"))
	}
}

func TestLoginOmitsSecureCookieOnPlainHTTP(t *testing.T) {
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

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if cookies[0].Secure {
		t.Fatal("plain HTTP dev and QA requests must receive a storable non-Secure cookie")
	}
}

func addAccessTokenCookie(req *http.Request, token string) {
	req.AddCookie(&http.Cookie{
		Name:  "accessToken",
		Value: "Bearer " + token,
	})
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
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kiteusers",
}

func newLoginTestUser(username string, passwordHash string, accessLevel int) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
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
