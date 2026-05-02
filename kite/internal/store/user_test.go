package store

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func TestUserStoreCRUD(t *testing.T) {
	ctx := context.Background()
	store := NewUserStore(fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteUserGVR: "KiteUserList",
	}))
	record := KiteUserRecord{
		Name: "test-user",
		Spec: KiteUserSpec{
			Username:     "test",
			Email:        "test@example.com",
			Password:     "hashed-password",
			Namespace:    "test-ns",
			ProfileImage: "base64",
			AccessLevel:  2,
		},
	}

	created, err := store.Create(ctx, record)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.GetName() != record.Name {
		t.Fatalf("expected created name %q, got %q", record.Name, created.GetName())
	}
	if created.GetNamespace() != "" {
		t.Fatalf("expected cluster-scoped KiteUser to have empty namespace, got %q", created.GetNamespace())
	}

	got, err := store.Get(ctx, record.Name)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	assertNestedString(t, got, "test@example.com", "spec", "email")
	assertNestedInt64(t, got, 2, "spec", "access_level")

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 user, got %d", len(list.Items))
	}

	record.Spec.Email = "updated@example.com"
	record.Spec.AccessLevel = 3
	updated, err := store.Update(ctx, record)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	assertNestedString(t, updated, "updated@example.com", "spec", "email")
	assertNestedInt64(t, updated, 3, "spec", "access_level")

	if err := store.Delete(ctx, record.Name); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	_, err = store.Get(ctx, record.Name)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func assertNestedString(t *testing.T, obj *unstructured.Unstructured, want string, fields ...string) {
	t.Helper()

	got, found, err := unstructured.NestedString(obj.Object, fields...)
	if err != nil {
		t.Fatalf("NestedString returned error: %v", err)
	}
	if !found {
		t.Fatalf("expected field %v to exist", fields)
	}
	if got != want {
		t.Fatalf("expected field %v to be %q, got %q", fields, want, got)
	}
}

func assertNestedInt64(t *testing.T, obj *unstructured.Unstructured, want int64, fields ...string) {
	t.Helper()

	got, found, err := unstructured.NestedInt64(obj.Object, fields...)
	if err != nil {
		t.Fatalf("NestedInt64 returned error: %v", err)
	}
	if !found {
		t.Fatalf("expected field %v to exist", fields)
	}
	if got != want {
		t.Fatalf("expected field %v to be %d, got %d", fields, want, got)
	}
}
