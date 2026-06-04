package apps

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func TestReconcileKiteUserDeletesPreviousNamespaceAfterNamespaceChange(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), userReconcileListKinds(),
		newUserReconcileKiteUser("ku-a", "new-ns", "old-ns"),
		newUserReconcileNamespace("old-ns"),
	)

	if err := ReconcileKiteUser(ctx, client, newUserReconcileKiteUser("ku-a", "new-ns", "old-ns")); err != nil {
		t.Fatalf("ReconcileKiteUser returned error: %v", err)
	}

	if _, err := client.Resource(namespaceGVR).Get(ctx, "old-ns", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected old namespace to be deleted, got %v", err)
	}
	if _, err := client.Resource(namespaceGVR).Get(ctx, "new-ns", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected new namespace to be created, got %v", err)
	}

	user, err := client.Resource(kiteUserGVR).Get(ctx, "ku-a", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected KiteUser to remain, got %v", err)
	}
	observedNamespace, _, _ := unstructured.NestedString(user.Object, "status", "observedNamespace")
	if observedNamespace != "new-ns" {
		t.Fatalf("expected observedNamespace new-ns, got %q", observedNamespace)
	}
}

func TestReconcileKiteUserBaseResourceRestoresDeletedNetworkPolicy(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), userReconcileListKinds(),
		newUserReconcileKiteUser("ku-a", "tenant-a", "tenant-a"),
	)

	deletedPolicy := newUserReconcileNetworkPolicy("tenant-a", userNetworkPolicyTenantIsolationEgress)
	if err := ReconcileKiteUserBaseResource(ctx, client, networkPolicyGVR, deletedPolicy); err != nil {
		t.Fatalf("ReconcileKiteUserBaseResource returned error: %v", err)
	}

	if _, err := client.Resource(networkPolicyGVR).Namespace("tenant-a").Get(ctx, userNetworkPolicyTenantIsolationEgress, metav1.GetOptions{}); err != nil {
		t.Fatalf("expected deleted NetworkPolicy to be restored, got %v", err)
	}
	if _, err := client.Resource(resourceQuotaGVR).Namespace("tenant-a").Get(ctx, userQuotaPolicyName, metav1.GetOptions{}); err != nil {
		t.Fatalf("expected ResourceQuota to be reconciled with base resources, got %v", err)
	}
}

func TestRestoreKiteNamespaceRecreatesNamespaceForReferencedKiteUser(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), userReconcileListKinds(),
		newUserReconcileKiteUser("ku-a", "tenant-a", "tenant-a"),
	)

	deletedNamespace := newUserReconcileNamespace("tenant-a")
	if err := RestoreKiteNamespace(ctx, client, deletedNamespace); err != nil {
		t.Fatalf("RestoreKiteNamespace returned error: %v", err)
	}

	if _, err := client.Resource(namespaceGVR).Get(ctx, "tenant-a", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected deleted namespace to be restored, got %v", err)
	}
}

// userReconcileListKinds returns fake dynamic client list kind mappings.
// The returned map supports KiteUser list calls during namespace cleanup.
// This helper is used by KiteUser reconcile unit tests.
func userReconcileListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		kiteUserGVR:      "KiteUserList",
		namespaceGVR:     "NamespaceList",
		networkPolicyGVR: "NetworkPolicyList",
		resourceQuotaGVR: "ResourceQuotaList",
	}
}

// newUserReconcileKiteUser creates a KiteUser object for namespace reconcile tests.
// name is metadata.name of the cluster-scoped KiteUser.
// namespace is the desired spec.namespace.
// observedNamespace is the previous status.observedNamespace value.
func newUserReconcileKiteUser(name string, namespace string, observedNamespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "anacnu.com/v1",
			"kind":       "KiteUser",
			"metadata": map[string]any{
				"name":       name,
				"generation": int64(2),
			},
			"spec": map[string]any{
				"username":      "user-a",
				"email":         "user-a@example.com",
				"password":      "hashed",
				"namespace":     namespace,
				"profile_image": "base64encodedimage",
				"access_level":  int64(0),
			},
			"status": map[string]any{
				"observedNamespace": observedNamespace,
			},
		},
	}
}

// newUserReconcileNamespace creates a Kite-managed Namespace test object.
// name is metadata.name of the Namespace.
// The returned object is safe for namespace cleanup because it carries the Kite managed label.
func newUserReconcileNamespace(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]any{
				"name": name,
				"labels": map[string]any{
					kiteNamespaceManagedByKey: kiteNamespaceManagedBy,
				},
			},
		},
	}
}

// newUserReconcileNetworkPolicy creates a minimal NetworkPolicy event object.
// namespace is metadata.namespace of the policy.
// name is metadata.name and should match one of the Kite-managed policy names.
func newUserReconcileNetworkPolicy(namespace string, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}
