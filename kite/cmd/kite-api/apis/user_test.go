package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR: "KiteUserList",
	}, newUserTestObject("user", "user", "user-ns", auth.AccessLevelUser))
	r := newUserTestRouterWithClient(t, tokenService, dynamicClient)
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	addAccessTokenCookie(req, userToken)
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

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR: "KiteUserList",
	}, newUserTestObject("manager", "manager", "manager-ns", auth.AccessLevelManager))
	r := newUserTestRouterWithClient(t, tokenService, dynamicClient)
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	addAccessTokenCookie(req, managerToken)
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
	if user.Username != "manager" || user.Email != "manager@gmail.com" || user.AccessLevel != auth.AccessLevelManager {
		t.Fatalf("unexpected user response: %+v", user)
	}
	if user.ProfileImage != "" {
		t.Fatalf("expected stored profile image to be omitted, got %q", user.ProfileImage)
	}
}

func TestSignUpCreatesFirstUserAsAdmin(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR: "KiteUserList",
	})
	r := newUserTestRouterWithClient(t, newTestTokenService(t), dynamicClient)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/signup",
		strings.NewReader(`{"username":"first","email":"first@example.com","password":"secret","profile_image":"malicious-profile-payload"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var res struct {
		User kiteUserResponse `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.User.AccessLevel != auth.AccessLevelAdmin {
		t.Fatalf("expected first user to be admin, got %d", res.User.AccessLevel)
	}
	if !strings.HasPrefix(res.User.Name, "ku-") {
		t.Fatalf("expected generated KiteUser name, got %+v", res.User)
	}
	if res.User.Namespace != "kite-user-"+res.User.Name {
		t.Fatalf("expected namespace derived from generated name, got %+v", res.User)
	}
	if res.User.ProfileImage != "" {
		t.Fatalf("expected empty profile image, got %+v", res.User)
	}

	created, err := dynamicClient.Resource(userTestGVR).Get(req.Context(), res.User.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read created user: %v", err)
	}
	password, _, err := unstructured.NestedString(created.Object, "spec", "password")
	if err != nil {
		t.Fatalf("failed to read stored password: %v", err)
	}
	if password == "secret" || password == "" {
		t.Fatalf("expected password to be stored as a non-empty one-way hash, got %q", password)
	}
	assertNestedString(t, created, "", "spec", "profile_image")
}

func TestSignUpCreatesLaterUserAsReadOnly(t *testing.T) {
	r := newUserTestRouter(t, newTestTokenService(t))
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/user",
		strings.NewReader(`{"username":"later","email":"later@example.com","password":"secret","namespace":"later-ns"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var res struct {
		User kiteUserResponse `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.User.AccessLevel != auth.AccessLevelReadOnly {
		t.Fatalf("expected later user to be read-only, got %d", res.User.AccessLevel)
	}
	if !strings.HasPrefix(res.User.Name, "ku-") {
		t.Fatalf("expected generated KiteUser name, got %+v", res.User)
	}
	if res.User.Namespace != "kite-user-"+res.User.Name {
		t.Fatalf("expected generated namespace, got %q", res.User.Namespace)
	}
}

func TestSignUpRejectsDuplicateEmail(t *testing.T) {
	r := newUserTestRouter(t, newTestTokenService(t))
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/signup",
		strings.NewReader(`{"username":"other","email":"test@gmail.com","password":"secret"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}
}

func TestUserUpdateIgnoresProfileImage(t *testing.T) {
	tokenService := newTestTokenService(t)
	adminToken, _, err := tokenService.IssueAccessToken("admin", auth.AccessLevelAdmin)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR: "KiteUserList",
	},
		newUserTestObject("admin", "admin", "admin-ns", auth.AccessLevelAdmin),
		newUserTestObject("target", "target", "target-ns", auth.AccessLevelUser),
	)
	r := newUserTestRouterWithClient(t, tokenService, dynamicClient)
	req := httptest.NewRequest(http.MethodPatch, "/api/users/target", strings.NewReader(`{"profile_image":"malicious-profile-payload"}`))
	req.Header.Set("Content-Type", "application/json")
	addAccessTokenCookie(req, adminToken)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res struct {
		User kiteUserResponse `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.User.ProfileImage != "" {
		t.Fatalf("expected profile image response to be empty, got %q", res.User.ProfileImage)
	}

	updated, err := dynamicClient.Resource(userTestGVR).Get(req.Context(), "target", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read updated user: %v", err)
	}
	assertNestedString(t, updated, "", "spec", "profile_image")
}

func newUserTestRouter(t *testing.T, tokenService *auth.TokenService) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	kiteUser := newUserTestObject("test", "test", "test", auth.AccessLevelAdmin)
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR: "KiteUserList",
	}, kiteUser)

	return newUserTestRouterWithClient(t, tokenService, dynamicClient)
}

func newUserTestRouterWithClient(t *testing.T, tokenService *auth.TokenService, dynamicClient dynamic.Interface) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	api := r.Group("/api")
	Register(api, Dependencies{
		Config: config.Config{
			PasswordSalt:   "test-salt",
			AccessTokenTTL: time.Hour,
		},
		TokenService:  tokenService,
		DynamicClient: dynamicClient,
	})

	return r
}

func newEmptyUserTestRouter(t *testing.T) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	api := r.Group("/api")
	Register(api, Dependencies{
		Config: config.Config{
			PasswordSalt:   "test-salt",
			AccessTokenTTL: time.Hour,
		},
		TokenService: newTestTokenService(t),
		DynamicClient: fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
			userTestGVR: "KiteUserList",
		}),
	})

	return r
}

func newUserTestObject(name string, username string, namespace string, accessLevel int) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteUser",
			"metadata": map[string]any{
				"name": name,
			},
			"spec": map[string]any{
				"username":      username,
				"email":         username + "@gmail.com",
				"password":      "hashed-password",
				"namespace":     namespace,
				"profile_image": "stored-profile-payload",
				"access_level":  int64(accessLevel),
			},
		},
	}
}

func newVirtualMachineTestObject(name string, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"cpu":        int64(1),
				"memory":     "1Gi",
				"image":      "ubuntu-22.04",
				"disk":       "5Gi",
				"powerState": "Off",
			},
		},
	}
}

var secretTestGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "secrets",
}

func newTestTokenService(t *testing.T) *auth.TokenService {
	t.Helper()

	tokenService, err := auth.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	return tokenService
}

func assertNestedInt64(t *testing.T, obj *unstructured.Unstructured, expected int64, fields ...string) {
	t.Helper()
	got, found, err := unstructured.NestedInt64(obj.Object, fields...)
	if err != nil {
		t.Fatalf("failed to read nested int64 %v: %v", fields, err)
	}
	if !found || got != expected {
		t.Fatalf("expected nested int64 %v to be %d, got %d found=%v", fields, expected, got, found)
	}
}

func assertNestedString(t *testing.T, obj *unstructured.Unstructured, expected string, fields ...string) {
	t.Helper()
	got, found, err := unstructured.NestedString(obj.Object, fields...)
	if err != nil {
		t.Fatalf("failed to read nested string %v: %v", fields, err)
	}
	if !found || got != expected {
		t.Fatalf("expected nested string %v to be %q, got %q found=%v", fields, expected, got, found)
	}
}

var userTestVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}
