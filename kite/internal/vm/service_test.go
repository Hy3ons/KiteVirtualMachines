package vm

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
				Namespace:   "tenant-a",
				Disk:        "20Gi",
				SSHID:       "asdf",
				SSHPassword: tt.password,
			})
			if err == nil {
				t.Fatalf("expected unsafe sshPassword %q to be rejected", tt.password)
			}
		})
	}
}

func TestCreateRecordAcceptsSafeSSHPassword(t *testing.T) {
	record, err := createRecord(CreateRequest{
		Namespace:   "tenant-a",
		Disk:        "20Gi",
		SSHID:       "asdf",
		SSHPassword: "pass word_123!",
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
	})

	service := NewService(dynamicClient, passwordSalt)
	if _, err := service.Create(ctx, CreateRequest{
		Name:        "vm-a",
		Namespace:   "tenant-a",
		Disk:        "20Gi",
		SSHID:       "asdf",
		SSHPassword: "pass word_123!",
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
}

func TestServiceCreateRejectsDuplicateSSHID(t *testing.T) {
	ctx := context.Background()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineTestGVR: "KiteVirtualMachineList",
	}, newTestKiteVirtualMachine("tenant-a", "vm-a", "asdf"))

	service := NewService(dynamicClient, "vm-password-salt")
	_, err := service.Create(ctx, CreateRequest{
		Name:        "vm-b",
		Namespace:   "tenant-b",
		Disk:        "20Gi",
		SSHID:       "asdf",
		SSHPassword: "pass word_123!",
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

func newTestKiteVirtualMachine(namespace string, name string, sshID string) *unstructured.Unstructured {
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
				"sshPasswordHash": auth.HashPassword("existing-password", "vm-password-salt"),
			},
		},
	}
}
