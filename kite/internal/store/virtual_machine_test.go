package store

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
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
