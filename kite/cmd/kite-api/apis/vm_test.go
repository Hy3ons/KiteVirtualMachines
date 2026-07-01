package apis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
	"kite/internal/config"
)

func TestVMUpdateRejectsSSHPasswordChange(t *testing.T) {
	tokenService := newTestTokenService(t)
	token, _, err := tokenService.IssueAccessToken("alice", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:               "KiteUserList",
		userTestVirtualMachineGVR: "KiteVirtualMachineList",
	}, newUserTestObject("alice", "alice", "alice-ns", auth.AccessLevelUser), newVirtualMachineTestObject("vm-a", "alice-ns"))

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	api := router.Group("/api")
	Register(api, Dependencies{
		Config: config.Config{
			PasswordSalt:   "test-salt",
			AccessTokenTTL: time.Hour,
		},
		TokenService:  tokenService,
		DynamicClient: dynamicClient,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/vms/vm-a", strings.NewReader(`{"sshPassword":"new password"}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "sshPassword cannot be changed after VM creation") {
		t.Fatalf("expected password immutability message, got %s", rec.Body.String())
	}
}
