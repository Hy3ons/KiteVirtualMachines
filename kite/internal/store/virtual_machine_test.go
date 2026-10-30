package store

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestVirtualMachineStoreCRUD(t *testing.T) {
	ctx := context.Background()
	store := NewVirtualMachineStore(fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineGVR: "KiteVirtualMachineList",
	}))
	record := KiteVirtualMachineRecord{
		Name:      "test-vm",
		Namespace: "test-ns",
		Spec: KiteVirtualMachineSpec{
			CPU:        2,
			Memory:     "4Gi",
			Image:      "ubuntu-22.04",
			Disk:       "20Gi",
			PowerState: "On",
		},
	}

	created, err := store.Create(ctx, record)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.GetName() != record.Name {
		t.Fatalf("expected created name %q, got %q", record.Name, created.GetName())
	}
	if created.GetNamespace() != record.Namespace {
		t.Fatalf("expected namespace %q, got %q", record.Namespace, created.GetNamespace())
	}

	got, err := store.Get(ctx, record.Namespace, record.Name)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	assertNestedInt64(t, got, 2, "spec", "cpu")
	assertNestedString(t, got, "On", "spec", "powerState")

	list, err := store.List(ctx, record.Namespace)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 VM, got %d", len(list.Items))
	}

	record.Spec.CPU = 4
	record.Spec.PowerState = "Off"
	updated, err := store.Update(ctx, record)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	assertNestedInt64(t, updated, 4, "spec", "cpu")
	assertNestedString(t, updated, "Off", "spec", "powerState")

	if err := store.Delete(ctx, record.Namespace, record.Name); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	_, err = store.Get(ctx, record.Namespace, record.Name)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

// TestVirtualMachineStoreUpdateRetriesStatusConflict simulates a controller status
// write between the API's read and update, and checks that power intent survives.
func TestVirtualMachineStoreUpdateRetriesStatusConflict(t *testing.T) {
	ctx := context.Background()
	record := KiteVirtualMachineRecord{Name: "test-vm", Namespace: "test-ns", Spec: KiteVirtualMachineSpec{CPU: 2, Memory: "4Gi", Disk: "20Gi", PowerState: "On"}}
	initial := newKiteVirtualMachineObject(record)
	initial.SetResourceVersion("1")
	client := fake.NewSimpleDynamicClient(runtime.NewScheme(), initial)
	attempts := 0
	client.PrependReactor("update", "kitevirtualmachines", func(action ktesting.Action) (bool, runtime.Object, error) {
		attempts++
		if attempts == 1 {
			observed := initial.DeepCopy()
			observed.SetResourceVersion("2")
			observed.SetFinalizers([]string{"controller-cleanup"})
			observed.Object["status"] = map[string]any{"phase": "Running"}
			if err := client.Tracker().Update(kiteVirtualMachineGVR, observed, record.Namespace); err != nil {
				t.Fatalf("simulate controller update: %v", err)
			}
			return true, nil, apierrors.NewConflict(kiteVirtualMachineGVR.GroupResource(), record.Name, nil)
		}
		next := action.(ktesting.UpdateAction).GetObject().(*unstructured.Unstructured)
		if next.GetResourceVersion() != "2" {
			t.Fatalf("retry used stale resourceVersion %q", next.GetResourceVersion())
		}
		return false, nil, nil
	})
	record.Spec.PowerState = "Off"
	updated, err := NewVirtualMachineStore(client).Update(ctx, record)
	if err != nil {
		t.Fatalf("power update failed after concurrent status write: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected conflict then successful update, got %d attempts", attempts)
	}
	assertNestedString(t, updated, "Off", "spec", "powerState")
	assertNestedString(t, updated, "Running", "status", "phase")
	if len(updated.GetFinalizers()) != 1 || updated.GetFinalizers()[0] != "controller-cleanup" {
		t.Fatalf("controller finalizers were lost: %v", updated.GetFinalizers())
	}
}

// TestVirtualMachineStoreUpdateDoesNotRetryForbidden checks that authorization
// failures propagate immediately instead of being treated as transient conflicts.
func TestVirtualMachineStoreUpdateDoesNotRetryForbidden(t *testing.T) {
	record := KiteVirtualMachineRecord{Name: "test-vm", Namespace: "test-ns"}
	client := fake.NewSimpleDynamicClient(runtime.NewScheme(), newKiteVirtualMachineObject(record))
	attempts := 0
	client.PrependReactor("update", "kitevirtualmachines", func(ktesting.Action) (bool, runtime.Object, error) {
		attempts++
		return true, nil, apierrors.NewForbidden(kiteVirtualMachineGVR.GroupResource(), record.Name, nil)
	})
	_, err := NewVirtualMachineStore(client).Update(context.Background(), record)
	if !apierrors.IsForbidden(err) || attempts != 1 {
		t.Fatalf("expected one forbidden update, got attempts=%d error=%v", attempts, err)
	}
	if _, err := client.Resource(kiteVirtualMachineGVR).Namespace(record.Namespace).Get(context.Background(), record.Name, metav1.GetOptions{}); err != nil {
		t.Fatalf("failed update removed existing VM: %v", err)
	}
}
