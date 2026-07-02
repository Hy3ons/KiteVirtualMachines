package apis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
)

var userTestVirtualMachineOfferGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachineoffers",
}

func TestAdminCreatesVMOfferForTargetUser(t *testing.T) {
	tokenService := newTestTokenService(t)
	adminToken, _, err := tokenService.IssueAccessToken("admin", auth.AccessLevelAdmin)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:                    "KiteUserList",
		userTestVirtualMachineOfferGVR: "KiteVirtualMachineOfferList",
	},
		newUserTestObject("admin", "admin", "admin-ns", auth.AccessLevelAdmin),
		newUserTestObject("alice", "alice", "alice-ns", auth.AccessLevelUser),
	)
	router := newUserTestRouterWithClient(t, tokenService, dynamicClient)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/vm-offers", strings.NewReader(`{"targetUser":"alice","cpu":4,"memory":"8Gi","disk":"30Gi","image":"ubuntu-22.04"}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, adminToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	list, err := dynamicClient.Resource(userTestVirtualMachineOfferGVR).Namespace("alice-ns").List(req.Context(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list offers: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(list.Items))
	}
	assertNestedInt64(t, &list.Items[0], 4, "spec", "cpu")
	assertNestedString(t, &list.Items[0], "8Gi", "spec", "memory")
	assertNestedString(t, &list.Items[0], "30Gi", "spec", "disk")
}

func TestUserClaimsVMOffer(t *testing.T) {
	tokenService := newTestTokenService(t)
	userToken, _, err := tokenService.IssueAccessToken("alice", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:                    "KiteUserList",
		userTestVirtualMachineGVR:      "KiteVirtualMachineList",
		userTestVirtualMachineOfferGVR: "KiteVirtualMachineOfferList",
		secretTestGVR:                  "SecretList",
	},
		newUserTestObject("alice", "alice", "alice-ns", auth.AccessLevelUser),
		newVirtualMachineOfferTestObject("offer-a", "alice-ns", time.Now().UTC().Add(time.Hour)),
	)
	router := newUserTestRouterWithClient(t, tokenService, dynamicClient)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vm-offers/offer-a/claim", strings.NewReader(`{"vmName":"vm-offered","sshId":"offered","initialLoginPassword":"secret-password","powerState":"On"}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, userToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	vm, err := dynamicClient.Resource(userTestVirtualMachineGVR).Namespace("alice-ns").Get(req.Context(), "vm-offered", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read claimed VM: %v", err)
	}
	assertNestedInt64(t, vm, 4, "spec", "cpu")
	assertNestedString(t, vm, "8Gi", "spec", "memory")
	assertNestedString(t, vm, "30Gi", "spec", "disk")
	if _, err := dynamicClient.Resource(userTestVirtualMachineOfferGVR).Namespace("alice-ns").Get(req.Context(), "offer-a", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected claimed offer to be deleted, got %v", err)
	}
}

func TestUserCannotClaimSameVMOfferTwice(t *testing.T) {
	tokenService := newTestTokenService(t)
	userToken, _, err := tokenService.IssueAccessToken("alice", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:                    "KiteUserList",
		userTestVirtualMachineGVR:      "KiteVirtualMachineList",
		userTestVirtualMachineOfferGVR: "KiteVirtualMachineOfferList",
		secretTestGVR:                  "SecretList",
	},
		newUserTestObject("alice", "alice", "alice-ns", auth.AccessLevelUser),
		newVirtualMachineOfferTestObject("offer-a", "alice-ns", time.Now().UTC().Add(time.Hour)),
	)
	router := newUserTestRouterWithClient(t, tokenService, dynamicClient)

	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/vm-offers/offer-a/claim", strings.NewReader(`{"vmName":"vm-offered","sshId":"offered","initialLoginPassword":"secret-password","powerState":"On"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(firstReq, userToken)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusCreated {
		t.Fatalf("expected first claim status %d, got %d: %s", http.StatusCreated, firstRec.Code, firstRec.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/vm-offers/offer-a/claim", strings.NewReader(`{"vmName":"vm-offered-again","sshId":"offered2","initialLoginPassword":"secret-password","powerState":"On"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(secondReq, userToken)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusNotFound {
		t.Fatalf("expected second claim status %d, got %d: %s", http.StatusNotFound, secondRec.Code, secondRec.Body.String())
	}
	list, err := dynamicClient.Resource(userTestVirtualMachineGVR).Namespace("alice-ns").List(secondReq.Context(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list VMs: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected one VM after repeated claim, got %d", len(list.Items))
	}
	if _, err := dynamicClient.Resource(userTestVirtualMachineGVR).Namespace("alice-ns").Get(secondReq.Context(), "vm-offered-again", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected second claim VM to be absent, got %v", err)
	}
}

func TestManagerCannotCreateVMOffer(t *testing.T) {
	tokenService := newTestTokenService(t)
	managerToken, _, err := tokenService.IssueAccessToken("manager", auth.AccessLevelManager)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:                    "KiteUserList",
		userTestVirtualMachineOfferGVR: "KiteVirtualMachineOfferList",
	}, newUserTestObject("manager", "manager", "manager-ns", auth.AccessLevelManager))
	router := newUserTestRouterWithClient(t, tokenService, dynamicClient)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/vm-offers", strings.NewReader(`{"targetNamespace":"alice-ns","cpu":4,"memory":"8Gi","disk":"30Gi"}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, managerToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func newVirtualMachineOfferTestObject(name string, namespace string, expiresAt time.Time) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachineOffer",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"cpu":       int64(4),
				"memory":    "8Gi",
				"image":     "ubuntu-22.04",
				"disk":      "30Gi",
				"expiresAt": expiresAt.Format(time.RFC3339),
				"createdBy": "admin",
			},
		},
	}
}
