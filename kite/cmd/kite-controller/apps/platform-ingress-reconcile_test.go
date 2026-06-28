package apps

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"kite/internal/config"
)

// TestReconcileKitePlatformIngressFromConfigMap_appliesHTTPIngressWhenRedirectCleanupFails verifies startup safety.
// Given runtime config leaves forceHttps disabled and redirect cleanup reports an error.
// When the platform ingress reconciler runs.
// Then the normal HTTP kite-platform Ingress is still applied for first-time admin access.
func TestReconcileKitePlatformIngressFromConfigMap_appliesHTTPIngressWhenRedirectCleanupFails(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), platformIngressReconcileListKinds(),
		newPlatformIngressRuntimeConfig(false),
	)
	appliedHTTPIngress := false
	client.Fake.PrependReactor("patch", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchAction, ok := action.(k8stesting.PatchAction)
		if ok && patchAction.GetName() == "kite-platform" {
			appliedHTTPIngress = true
		}
		return true, newPlatformIngressObject("kite-platform"), nil
	})
	client.Fake.PrependReactor("delete", "middlewares", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("middleware API unavailable")
	})

	err := ReconcileKitePlatformIngressFromConfigMap(ctx, client)

	if err != nil {
		t.Fatalf("expected redirect middleware cleanup failure not to block HTTP ingress, got %v", err)
	}
	if !appliedHTTPIngress {
		t.Fatalf("expected kite-platform ingress to be applied")
	}
}

// platformIngressReconcileListKinds returns fake dynamic client list kind mappings.
// The returned map covers runtime ConfigMaps, networking Ingresses, and Traefik middleware.
// This helper is used by platform ingress reconcile unit tests.
func platformIngressReconcileListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		configMapGVR:         "ConfigMapList",
		ingressGVR:           "IngressList",
		traefikMiddlewareGVR: "MiddlewareList",
	}
}

// newPlatformIngressRuntimeConfig creates the kite-runtime-config test object.
// forceHTTPS controls the forceHttps data value used by the platform ingress reconciler.
// The returned ConfigMap is used by tests that exercise HTTP and HTTPS ingress modes.
func newPlatformIngressRuntimeConfig(forceHTTPS bool) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      kiteGlobalConfigName,
				"namespace": kiteGlobalConfigNamespace,
			},
			"data": map[string]any{
				kiteGlobalBaseDomainKey:    "hy3on.site",
				config.ForceHTTPSConfigKey: fmt.Sprintf("%t", forceHTTPS),
			},
		},
	}
}

// newPlatformIngressObject creates a minimal networking.k8s.io Ingress test object.
// name controls metadata.name in the kite namespace.
// The returned object lets fake dynamic clients exercise patch-based reconcile updates.
func newPlatformIngressObject(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "Ingress",
			"metadata": map[string]any{
				"name":      name,
				"namespace": kiteGlobalConfigNamespace,
			},
		},
	}
}
