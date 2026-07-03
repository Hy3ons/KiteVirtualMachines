package offer

import (
	"context"
	"errors"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	kubetesting "k8s.io/client-go/testing"
)

var testOfferGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachineoffers",
}

var testVMGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}

func TestClaimReturnsConflictWithoutCreatingVMWhenStatusUpdateConflicts(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		testOfferGVR: "KiteVirtualMachineOfferList",
		testVMGVR:    "KiteVirtualMachineList",
	}, newOfferTestObject("offer-a", "alice-ns", phaseAvailable))
	client.PrependReactor("update", "kitevirtualmachineoffers", func(action kubetesting.Action) (bool, runtime.Object, error) {
		updateAction, ok := action.(kubetesting.UpdateAction)
		if ok && updateAction.GetSubresource() == "status" {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Group: "hy3ons.github.io", Resource: "kitevirtualmachineoffers"}, "offer-a", errors.New("stale resource version"))
		}
		return false, nil, nil
	})
	service := NewService(client, "test-salt")

	_, err := service.Claim(ctx, "alice-ns", "offer-a", "alice", ClaimRequest{
		VMName:               "vm-offered",
		SSHID:                "ubuntu",
		InitialLoginPassword: "secret-password",
		PowerState:           "On",
	})

	if kind, ok := RequestErrorKind(err); !ok || kind != ErrorKindConflict {
		t.Fatalf("expected conflict request error, got %v", err)
	}
	vms, err := client.Resource(testVMGVR).Namespace("alice-ns").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list VMs: %v", err)
	}
	if len(vms.Items) != 0 {
		t.Fatalf("expected no VM to be created after claim conflict, got %d", len(vms.Items))
	}
}

func TestClaimRejectsAlreadyClaimedOfferWithoutCreatingVM(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		testOfferGVR: "KiteVirtualMachineOfferList",
		testVMGVR:    "KiteVirtualMachineList",
	}, newOfferTestObject("offer-a", "alice-ns", phaseClaimed))
	service := NewService(client, "test-salt")

	_, err := service.Claim(ctx, "alice-ns", "offer-a", "alice", ClaimRequest{
		VMName:               "vm-offered",
		SSHID:                "ubuntu",
		InitialLoginPassword: "secret-password",
		PowerState:           "On",
	})

	if kind, ok := RequestErrorKind(err); !ok || kind != ErrorKindConflict {
		t.Fatalf("expected conflict request error, got %v", err)
	}
	vms, err := client.Resource(testVMGVR).Namespace("alice-ns").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list VMs: %v", err)
	}
	if len(vms.Items) != 0 {
		t.Fatalf("expected already claimed offer to create no VM, got %d", len(vms.Items))
	}
}

func newOfferTestObject(name string, namespace string, phase string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
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
				"expiresAt": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
				"createdBy": "admin",
			},
		},
	}
	if phase != "" {
		obj.Object["status"] = map[string]any{"phase": phase}
	}
	return obj
}
