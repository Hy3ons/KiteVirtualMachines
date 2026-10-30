package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kite/internal/auth"
)

func TestSessionMetadataMatchesSignedExpiryAndCookieLifetime(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d", rec.Code)
	}
	var login loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	cookie := rec.Result().Cookies()[0]
	claims, err := newTestTokenService(t).VerifyAccessToken(cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	if login.ExpiresIn != 3600 || cookie.MaxAge != 3600 || claims.ExpiresAt-claims.IssuedAt != 3600 {
		t.Fatal("HTTP cookie, JWT and login metadata must all expire after sixty minutes")
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var me struct {
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || me.ExpiresAt != login.ExpiresAt {
		t.Fatal("/me must expose the same verified expiry as login")
	}
}

func TestExpiredSignedTokenIsRejectedAndCookieCleared(t *testing.T) {
	service, err := auth.NewTokenService("test-secret", time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := service.IssueAccessToken("admin", auth.AccessLevelAdmin)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	addAccessTokenCookie(req, token)
	rec := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatal("expired JWT must return 401 and remove the session cookie")
	}
}
