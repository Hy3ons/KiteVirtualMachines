package apis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

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

func TestRequireAccessLevelRejectsStaleAdminTokenWhenCurrentUserWasDemoted(t *testing.T) {
	// Given
	tokenService := newTestTokenService(t)
	staleAdminToken, _, err := tokenService.IssueAccessToken("demoted", auth.AccessLevelAdmin)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR: "KiteUserList",
	}, newUserTestObject("demoted", "demoted", "demoted-ns", auth.AccessLevelReadOnly))
	r := newUserTestRouterWithClient(t, tokenService, dynamicClient)
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	addAccessTokenCookie(req, staleAdminToken)
	rec := httptest.NewRecorder()

	// When
	r.ServeHTTP(rec, req)

	// Then
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}
