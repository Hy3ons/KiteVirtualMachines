package apis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
)

func TestManagerCanChangeOnlyReadOnlyAndUserLevels(t *testing.T) {
	tokenService := newTestTokenService(t)
	managerToken, _, err := tokenService.IssueAccessToken("manager", auth.AccessLevelManager)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR: "KiteUserList",
	},
		newUserTestObject("manager", "manager", "manager-ns", auth.AccessLevelManager),
		newUserTestObject("target", "target", "target-ns", auth.AccessLevelReadOnly),
		newUserTestObject("admin", "admin", "admin-ns", auth.AccessLevelAdmin),
	)
	r := newUserTestRouterWithClient(t, tokenService, dynamicClient)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/target/access-level", strings.NewReader(`{"access_level":1}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, managerToken)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/target/access-level", strings.NewReader(`{"access_level":2}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, managerToken)
	rec = httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/admin/access-level", strings.NewReader(`{"access_level":1}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, managerToken)
	rec = httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func TestManagerCannotUseGlobalVMControl(t *testing.T) {
	tokenService := newTestTokenService(t)
	managerToken, _, err := tokenService.IssueAccessToken("manager", auth.AccessLevelManager)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:               "KiteUserList",
		userTestVirtualMachineGVR: "KiteVirtualMachineList",
	},
		newUserTestObject("manager", "manager", "manager-ns", auth.AccessLevelManager),
		newVirtualMachineTestObject("vm-a", "target-ns"),
	)
	r := newUserTestRouterWithClient(t, tokenService, dynamicClient)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/vms", nil)
	addAccessTokenCookie(req, managerToken)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func TestAdminDeleteUserDeletesChildVirtualMachines(t *testing.T) {
	tokenService := newTestTokenService(t)
	adminToken, _, err := tokenService.IssueAccessToken("admin", auth.AccessLevelAdmin)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:               "KiteUserList",
		userTestVirtualMachineGVR: "KiteVirtualMachineList",
	}, newUserTestObject("target", "target", "target-ns", auth.AccessLevelUser), newVirtualMachineTestObject("target-vm", "target-ns"))
	r := newUserTestRouterWithClient(t, tokenService, dynamicClient)

	req := httptest.NewRequest(http.MethodDelete, "/api/users/target", nil)
	addAccessTokenCookie(req, adminToken)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	_, err = dynamicClient.Resource(userTestVirtualMachineGVR).Namespace("target-ns").Get(req.Context(), "target-vm", metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected child VM CRD to be deleted, got %v", err)
	}
}
