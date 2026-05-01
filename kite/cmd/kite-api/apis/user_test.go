package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
	"kite/internal/config"
)

func TestUserListRequiresManagerAccess(t *testing.T) {
	tokenService := newTestTokenService(t)
	userToken, _, err := tokenService.IssueAccessToken("user", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	r := newUserTestRouter(t, tokenService)
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUserListReturnsKiteUsers(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res struct {
		Users []kiteUserResponse `json:"users"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(res.Users))
	}

	user := res.Users[0]
	if user.Username != "test" || user.Email != "test@gmail.com" || user.AccessLevel != 3 {
		t.Fatalf("unexpected user response: %+v", user)
	}
}

func newUserTestRouter(t *testing.T, tokenService *auth.TokenService) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	kiteUser := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "anacnu.com/v1",
			"kind":       "KiteUser",
			"metadata": map[string]any{
				"name": "test",
			},
			"spec": map[string]any{
				"username":      "test",
				"email":         "test@gmail.com",
				"password":      "hashed-password",
				"namespace":     "test",
				"profile_image": "base64encodedimage",
				"access_level":  int64(3),
			},
		},
	}

	r := gin.Default()
	api := r.Group("/api")
	Register(api, Dependencies{
		Config: config.Config{
			AccessTokenTTL: time.Hour,
		},
		TokenService:  tokenService,
		DynamicClient: fake.NewSimpleDynamicClient(runtime.NewScheme(), kiteUser),
	})

	return r
}

func newTestTokenService(t *testing.T) *auth.TokenService {
	t.Helper()

	tokenService, err := auth.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	return tokenService
}
