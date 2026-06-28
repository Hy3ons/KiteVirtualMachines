package apis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
)

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
