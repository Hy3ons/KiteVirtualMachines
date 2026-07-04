package vm

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"

	"kite/internal/auth"
	"kite/internal/guestlogin"
)

var kiteVirtualMachineTestGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}

func TestCreateRecordRejectsUnsafeSSHPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "leading space", password: " password"},
		{name: "trailing space", password: "password "},
		{name: "newline", password: "pass\nword"},
		{name: "carriage return", password: "pass\rword"},
		{name: "colon", password: "pass:word"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createRecord(CreateRequest{
				Namespace:    "tenant-a",
				Disk:         "20Gi",
				DomainPrefix: "app-a",
				SSHID:        "asdf",
				SSHPassword:  tt.password,
			})
			if err == nil {
				t.Fatalf("expected unsafe sshPassword %q to be rejected", tt.password)
			}
		})
	}
}

func TestCreateRecordAcceptsSafeSSHPassword(t *testing.T) {
	record, err := createRecord(CreateRequest{
		Namespace:    "tenant-a",
		Disk:         "20Gi",
		DomainPrefix: "app-a",
		SSHID:        "asdf",
		SSHPassword:  "pass word_123!",
	})
	if err != nil {
		t.Fatalf("expected safe sshPassword to be accepted, got %v", err)
	}
	if record.Spec.SSHPasswordHash != "" {
		t.Fatalf("createRecord must not store plain or hashed password, got %q", record.Spec.SSHPasswordHash)
	}
}

func TestServiceCreateStoresSSHPasswordHash(t *testing.T) {
	ctx := context.Background()
	passwordSalt := "vm-password-salt"
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineTestGVR: "KiteVirtualMachineList",
		secretGVR:                 "SecretList",
	})

	service := NewService(dynamicClient, passwordSalt)
	if _, err := service.Create(ctx, CreateRequest{
		Name:         "vm-a",
		Namespace:    "tenant-a",
		Disk:         "20Gi",
		DomainPrefix: "app-a",
		SSHID:        "asdf",
		SSHPassword:  "pass word_123!",
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	obj, err := dynamicClient.Resource(kiteVirtualMachineTestGVR).Namespace("tenant-a").Get(ctx, "vm-a", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read created VM: %v", err)
	}
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if _, exists := spec["sshPassword"]; exists {
		t.Fatalf("spec.sshPassword must not be stored: %#v", spec)
	}
	passwordHash, _, _ := unstructured.NestedString(obj.Object, "spec", "sshPasswordHash")
	if passwordHash == "" || passwordHash == "pass word_123!" {
		t.Fatalf("expected hashed password, got %q", passwordHash)
	}
	if !auth.VerifyPassword("pass word_123!", passwordSalt, passwordHash) {
		t.Fatalf("stored password hash does not verify")
	}

	secret, err := dynamicClient.Resource(secretGVR).Namespace("tenant-a").Get(ctx, guestlogin.SecretName("vm-a"), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read guest login Secret: %v", err)
	}
	encodedHash, _, _ := unstructured.NestedString(secret.Object, "data", guestlogin.PasswordHashKey)
	guestPasswordHash, err := base64.StdEncoding.DecodeString(encodedHash)
	if err != nil {
		t.Fatalf("guest login Secret contains invalid base64 hash: %v", err)
	}
	if string(guestPasswordHash) == "" || string(guestPasswordHash) == "pass word_123!" {
		t.Fatalf("expected guest password hash, got %q", string(guestPasswordHash))
	}
	if !guestlogin.VerifyPasswordHash("pass word_123!", string(guestPasswordHash)) {
		t.Fatalf("guest password hash does not verify")
	}
}

func TestServiceCreateRollsBackVMWhenGuestLoginSecretFails(t *testing.T) {
	ctx := context.Background()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineTestGVR: "KiteVirtualMachineList",
		secretGVR:                 "SecretList",
	})
	dynamicClient.PrependReactor("create", "secrets", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("secret create failed")
	})

	service := NewService(dynamicClient, "vm-password-salt")
	_, err := service.Create(ctx, CreateRequest{
		Name:         "vm-a",
		Namespace:    "tenant-a",
		Disk:         "20Gi",
		DomainPrefix: "app-a",
		SSHID:        "asdf",
		SSHPassword:  "pass word_123!",
	})
	if err == nil {
		t.Fatal("expected Create to fail when guest login Secret creation fails")
	}
	if _, err := dynamicClient.Resource(kiteVirtualMachineTestGVR).Namespace("tenant-a").Get(ctx, "vm-a", metav1.GetOptions{}); !IsNotFound(err) {
		t.Fatalf("expected created VM to be rolled back, got %v", err)
	}
}

func TestServiceUpdateRejectsSSHPasswordChange(t *testing.T) {
	ctx := context.Background()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineTestGVR: "KiteVirtualMachineList",
	}, newTestKiteVirtualMachine("tenant-a", "vm-a", "asdf", "app-a"))

	nextPassword := "new password"
	service := NewService(dynamicClient, "vm-password-salt")
	_, err := service.Update(ctx, "tenant-a", "vm-a", UpdateRequest{
		SSHPassword: &nextPassword,
	})
	if err == nil {
		t.Fatal("expected password update to be rejected")
	}
	requestErr, ok := err.(RequestError)
	if !ok {
		t.Fatalf("expected RequestError, got %T: %v", err, err)
	}
	if requestErr.Kind != ErrorKindInvalid {
		t.Fatalf("expected invalid error, got %s: %s", requestErr.Kind, requestErr.Message)
	}
}

func TestServiceCreateRejectsDuplicateSSHID(t *testing.T) {
	ctx := context.Background()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineTestGVR: "KiteVirtualMachineList",
	}, newTestKiteVirtualMachine("tenant-a", "vm-a", "asdf", "app-a"))

	service := NewService(dynamicClient, "vm-password-salt")
	_, err := service.Create(ctx, CreateRequest{
		Name:         "vm-b",
		Namespace:    "tenant-b",
		Disk:         "20Gi",
		DomainPrefix: "app-b",
		SSHID:        "asdf",
		SSHPassword:  "pass word_123!",
	})
	if err == nil {
		t.Fatal("expected duplicate sshId to be rejected")
	}
	requestErr, ok := err.(RequestError)
	if !ok {
		t.Fatalf("expected RequestError, got %T: %v", err, err)
	}
	if requestErr.Kind != ErrorKindConflict {
		t.Fatalf("expected conflict error, got %s: %s", requestErr.Kind, requestErr.Message)
	}
}

func TestServiceCreateRejectsDuplicateDomainPrefixAcrossNamespaces(t *testing.T) {
	ctx := context.Background()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineTestGVR: "KiteVirtualMachineList",
	}, newTestKiteVirtualMachine("tenant-a", "vm-a", "ssh-a", "shared-app"))

	service := NewService(dynamicClient, "vm-password-salt")
	_, err := service.Create(ctx, CreateRequest{
		Name:         "vm-b",
		Namespace:    "tenant-b",
		Disk:         "20Gi",
		DomainPrefix: "shared-app",
		SSHID:        "ssh-b",
		SSHPassword:  "pass word_123!",
	})
	if err == nil {
		t.Fatal("expected duplicate domainPrefix to be rejected")
	}
	requestErr, ok := err.(RequestError)
	if !ok {
		t.Fatalf("expected RequestError, got %T: %v", err, err)
	}
	if requestErr.Kind != ErrorKindConflict {
		t.Fatalf("expected conflict error, got %s: %s", requestErr.Kind, requestErr.Message)
	}
}

func TestServiceUpdateRejectsReservedDomainPrefix(t *testing.T) {
	ctx := context.Background()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineTestGVR: "KiteVirtualMachineList",
	}, newTestKiteVirtualMachine("tenant-a", "vm-a", "ssh-a", "app-a"))

	nextDomainPrefix := "api"
	service := NewService(dynamicClient, "vm-password-salt")
	_, err := service.Update(ctx, "tenant-a", "vm-a", UpdateRequest{
		DomainPrefix: &nextDomainPrefix,
	})
	if err == nil {
		t.Fatal("expected reserved domainPrefix to be rejected")
	}
	requestErr, ok := err.(RequestError)
	if !ok {
		t.Fatalf("expected RequestError, got %T: %v", err, err)
	}
	if requestErr.Kind != ErrorKindInvalid {
		t.Fatalf("expected invalid error, got %s: %s", requestErr.Kind, requestErr.Message)
	}
}

func TestCreateRecordRejectsInvalidDomainPrefix(t *testing.T) {
	tests := []struct {
		name         string
		domainPrefix string
	}{
		{name: "uppercase", domainPrefix: "BadApp"},
		{name: "underscore", domainPrefix: "bad_app"},
		{name: "leading hyphen", domainPrefix: "-bad"},
		{name: "trailing hyphen", domainPrefix: "bad-"},
		{name: "reserved", domainPrefix: "www"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createRecord(CreateRequest{
				Namespace:    "tenant-a",
				Disk:         "20Gi",
				DomainPrefix: tt.domainPrefix,
				SSHID:        "ssh-a",
				SSHPassword:  "pass word_123!",
			})
			if err == nil {
				t.Fatalf("expected invalid domainPrefix %q to be rejected", tt.domainPrefix)
			}
		})
	}
}

func TestCreateRecordRejectsMissingDomainPrefix(t *testing.T) {
	_, err := createRecord(CreateRequest{
		Namespace:   "tenant-a",
		Disk:        "20Gi",
		SSHID:       "ssh-a",
		SSHPassword: "pass word_123!",
	})
	if err == nil {
		t.Fatal("expected missing domainPrefix to be rejected")
	}
	requestErr, ok := err.(RequestError)
	if !ok {
		t.Fatalf("expected RequestError, got %T: %v", err, err)
	}
	if requestErr.Kind != ErrorKindInvalid {
		t.Fatalf("expected invalid error, got %s: %s", requestErr.Kind, requestErr.Message)
	}
	if requestErr.Message != "domainPrefix is required" {
		t.Fatalf("expected domainPrefix required message, got %q", requestErr.Message)
	}
}

func newTestKiteVirtualMachine(namespace string, name string, sshID string, domainPrefix string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"sshId":           sshID,
				"domainPrefix":    domainPrefix,
				"sshPasswordHash": auth.LegacyHashPassword("existing-password", "vm-password-salt"),
			},
		},
	}
}
