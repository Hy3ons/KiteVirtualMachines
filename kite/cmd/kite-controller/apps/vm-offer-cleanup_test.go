package apps

import (
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func TestDeleteExpiredKiteVirtualMachineOffersRemovesOnlyExpiredOffers(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		kiteVirtualMachineOfferGVR: "KiteVirtualMachineOfferList",
	},
		newOfferCleanupTestObject("expired", "tenant-a", now.Add(-time.Minute)),
		newOfferCleanupTestObject("active", "tenant-a", now.Add(time.Hour)),
	)

	if err := DeleteExpiredKiteVirtualMachineOffers(t.Context(), dynamicClient, now); err != nil {
		t.Fatalf("DeleteExpiredKiteVirtualMachineOffers returned error: %v", err)
	}

	if _, err := dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace("tenant-a").Get(t.Context(), "expired", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected expired offer to be deleted, got %v", err)
	}
	if _, err := dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace("tenant-a").Get(t.Context(), "active", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected active offer to remain, got %v", err)
	}
}

func newOfferCleanupTestObject(name string, namespace string, expiresAt time.Time) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachineOffer",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"cpu":       int64(2),
				"memory":    "4Gi",
				"disk":      "25Gi",
				"image":     "ubuntu-22.04",
				"expiresAt": expiresAt.Format(time.RFC3339),
				"createdBy": "admin",
			},
		},
	}
}
