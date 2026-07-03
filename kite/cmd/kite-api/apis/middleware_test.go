package apis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kite/internal/auth"
)

func TestRequireAccessLevelRejectsAuthorizationHeaderWithoutCookie(t *testing.T) {
	tokenService := newTestTokenService(t)
	managerToken, _, err := tokenService.IssueAccessToken("manager", auth.AccessLevelManager)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	r := newUserTestRouter(t, tokenService)
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+managerToken)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
