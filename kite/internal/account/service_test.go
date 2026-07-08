package account

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
)

var accountTestUserGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kiteusers",
}

func TestAuthenticateMigratesLegacyPasswordHashToBcrypt(t *testing.T) {
	ctx := context.Background()
	passwordSalt := "account-test-salt"
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		accountTestUserGVR: "KiteUserList",
	}, newAccountTestUser("alice", "alice@example.com", auth.LegacyHashPassword("secret-password", passwordSalt)))

	service := NewService(dynamicClient, passwordSalt)
	user, ok, err := service.Authenticate(ctx, "alice@example.com", "secret-password")
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected legacy password to authenticate")
	}
	if user.Username != "alice" {
		t.Fatalf("expected alice, got %+v", user)
	}

	updated, err := dynamicClient.Resource(accountTestUserGVR).Get(ctx, "alice", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read updated user: %v", err)
	}
	passwordHash, _, _ := unstructured.NestedString(updated.Object, "spec", "password")
	if passwordHash == "" || auth.PasswordNeedsRehash(passwordHash) {
		t.Fatalf("expected migrated bcrypt hash, got %q", passwordHash)
	}
	if !auth.VerifyPassword("secret-password", passwordSalt, passwordHash) {
		t.Fatal("expected migrated hash to verify")
	}
}

func TestSignUpRejectsDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	passwordSalt := "account-test-salt"
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		accountTestUserGVR: "KiteUserList",
	}, newAccountTestUser("alice", "shared@example.com", auth.LegacyHashPassword("secret-password", passwordSalt)))

	service := NewService(dynamicClient, passwordSalt)
	_, err := service.SignUp(ctx, SignUpRequest{
		Username: "bob",
		Email:    "SHARED@example.com",
		Password: "secret-password",
	})

	if err == nil {
		t.Fatal("expected duplicate email to be rejected")
	}
	if kind, ok := RequestErrorKind(err); !ok || kind != ErrorKindConflict {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestUpdateRejectsDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	passwordSalt := "account-test-salt"
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		accountTestUserGVR: "KiteUserList",
	},
		newAccountTestUser("alice", "alice@example.com", auth.LegacyHashPassword("secret-password", passwordSalt)),
		newAccountTestUser("bob", "bob@example.com", auth.LegacyHashPassword("secret-password", passwordSalt)),
	)

	service := NewService(dynamicClient, passwordSalt)
	duplicateEmail := "ALICE@example.com"
	_, err := service.Update(ctx, "bob", UpdateRequest{Email: &duplicateEmail})

	if err == nil {
		t.Fatal("expected duplicate email update to be rejected")
	}
	if kind, ok := RequestErrorKind(err); !ok || kind != ErrorKindConflict {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestListOmitsStoredProfileImage(t *testing.T) {
	ctx := context.Background()
	passwordSalt := "account-test-salt"
	user := newAccountTestUser("alice", "alice@example.com", auth.LegacyHashPassword("secret-password", passwordSalt))
	if err := unstructured.SetNestedField(user.Object, "malicious-profile-payload", "spec", "profile_image"); err != nil {
		t.Fatalf("failed to set profile_image fixture: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		accountTestUserGVR: "KiteUserList",
	}, user)

	service := NewService(dynamicClient, passwordSalt)
	users, err := service.List(ctx)

	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].ProfileImage != "" {
		t.Fatalf("expected stored profile image to be omitted, got %q", users[0].ProfileImage)
	}
}

func TestUpdateIgnoresProfileImageInput(t *testing.T) {
	ctx := context.Background()
	passwordSalt := "account-test-salt"
	user := newAccountTestUser("alice", "alice@example.com", auth.LegacyHashPassword("secret-password", passwordSalt))
	if err := unstructured.SetNestedField(user.Object, "legacy-profile-payload", "spec", "profile_image"); err != nil {
		t.Fatalf("failed to set profile_image fixture: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		accountTestUserGVR: "KiteUserList",
	}, user)

	service := NewService(dynamicClient, passwordSalt)
	maliciousProfileImage := "data:image/svg+xml,<svg onload=alert(1)>"
	updatedUser, err := service.Update(ctx, "alice", UpdateRequest{ProfileImage: &maliciousProfileImage})

	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updatedUser.ProfileImage != "" {
		t.Fatalf("expected profile image response to be empty, got %q", updatedUser.ProfileImage)
	}

	updatedObject, err := dynamicClient.Resource(accountTestUserGVR).Get(ctx, "alice", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read updated user: %v", err)
	}
	profileImage, found, err := unstructured.NestedString(updatedObject.Object, "spec", "profile_image")
	if err != nil {
		t.Fatalf("failed to read stored profile_image: %v", err)
	}
	if !found || profileImage != "" {
		t.Fatalf("expected stored profile_image to be empty, got %q found=%v", profileImage, found)
	}
}

func newAccountTestUser(name string, email string, passwordHash string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteUser",
			"metadata": map[string]any{
				"name": name,
			},
			"spec": map[string]any{
				"username":      name,
				"email":         email,
				"password":      passwordHash,
				"namespace":     "kite-user-" + name,
				"profile_image": defaultProfileImage,
				"access_level":  int64(auth.AccessLevelUser),
			},
		},
	}
}
