package apis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestVMCreateRejectsLevelOneUser_whenQuotaReached(t *testing.T) {
	tokenService := newTestTokenService(t)
	token, _, err := tokenService.IssueAccessToken("alice", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:               "KiteUserList",
		userTestVirtualMachineGVR: "KiteVirtualMachineList",
	}, newUserTestObject("alice", "alice", "alice-ns", auth.AccessLevelUser),
		newVirtualMachineTestObject("vm-a", "alice-ns"),
		newVirtualMachineTestObject("vm-b", "alice-ns"),
		newVirtualMachineTestObject("vm-c", "alice-ns"))

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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vms", strings.NewReader(`{"name":"vm-d","cpu":8,"memory":"16Gi","image":"ubuntu-22.04","disk":"80Gi","sshId":"vm-d","sshPassword":"secret-password","powerState":"Off"}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Level 1 users can create up to 3 virtual machines") {
		t.Fatalf("expected quota message, got %s", rec.Body.String())
	}

	list, err := dynamicClient.Resource(userTestVirtualMachineGVR).Namespace("alice-ns").List(t.Context(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list virtual machines after rejected create: %v", err)
	}
	if len(list.Items) != levelOneVMQuota {
		t.Fatalf("expected quota rejection to keep %d VMs, got %d", levelOneVMQuota, len(list.Items))
	}
	for _, item := range list.Items {
		if item.GetName() == "vm-d" {
			t.Fatalf("expected rejected VM not to be created")
		}
	}
}
