package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"kite/internal/config"
	"kite/internal/platform"
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

// TestReconcileKitePlatformIngressFromConfigMap_deletesIngressWhenBaseDomainIsEmpty verifies hostless ingress prevention.
// Given runtime config has no baseDomain.
// When the platform ingress reconciler runs.
// Then Kite-managed external Ingress resources are deleted instead of rendered without host matching.
func TestReconcileKitePlatformIngressFromConfigMap_deletesIngressWhenBaseDomainIsEmpty(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), platformIngressReconcileListKinds(),
		newPlatformIngressRuntimeConfigWithBaseDomain("", false),
	)
	deletedPlatformIngress := false
	client.Fake.PrependReactor("delete", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteAction, ok := action.(k8stesting.DeleteAction)
		if ok && deleteAction.GetName() == "kite-platform" {
			deletedPlatformIngress = true
		}
		return true, nil, nil
	})
	client.Fake.PrependReactor("delete", "middlewares", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})

	err := ReconcileKitePlatformIngressFromConfigMap(ctx, client)

	if err != nil {
		t.Fatalf("expected empty baseDomain cleanup to succeed, got %v", err)
	}
	if !deletedPlatformIngress {
		t.Fatalf("expected kite-platform ingress to be deleted")
	}
}

// TestReconcileKitePlatformIngressFromConfigMap_rejectsForceHTTPSWithoutTLS verifies HTTPS safety.
// Given forceHttps is enabled but no TLS Secret exists.
// When the platform ingress reconciler runs.
// Then reconcile fails before applying a misleading HTTPS redirect.
func TestReconcileKitePlatformIngressFromConfigMap_rejectsForceHTTPSWithoutTLS(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), platformIngressReconcileListKinds(),
		newPlatformIngressRuntimeConfig(true),
	)

	err := ReconcileKitePlatformIngressFromConfigMap(ctx, client)

	if err == nil {
		t.Fatalf("expected forceHttps without TLS to fail")
	}
}

// TestReconcileKitePlatformIngressFromConfigMap_usesTLSSecretWhenPresent verifies TLS wiring.
// Given forceHttps is enabled and kite/global-tls-secret exists.
// When the platform ingress reconciler applies kite-platform.
// Then the rendered Ingress references the namespace-local TLS Secret.
func TestReconcileKitePlatformIngressFromConfigMap_usesTLSSecretWhenPresent(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), platformIngressReconcileListKinds(),
		newPlatformIngressRuntimeConfig(true),
		newPlatformIngressTLSSecret(),
	)
	platformIngressUsesTLS := false
	client.Fake.PrependReactor("patch", "middlewares", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, newPlatformHTTPSRedirectMiddlewareObject(), nil
	})
	client.Fake.PrependReactor("patch", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchAction, ok := action.(k8stesting.PatchAction)
		if !ok || patchAction.GetName() != "kite-platform" {
			return true, newPlatformIngressObject("kite-platform-http-redirect"), nil
		}
		var patched map[string]any
		if err := json.Unmarshal(patchAction.GetPatch(), &patched); err != nil {
			t.Fatalf("failed to parse ingress patch: %v", err)
		}
		spec, _ := patched["spec"].(map[string]any)
		tlsEntries, _ := spec["tls"].([]any)
		if len(tlsEntries) == 1 {
			tlsEntry, _ := tlsEntries[0].(map[string]any)
			platformIngressUsesTLS = tlsEntry["secretName"] == platform.GlobalTLSSecretName
		}
		return true, newPlatformIngressObject("kite-platform"), nil
	})

	err := ReconcileKitePlatformIngressFromConfigMap(ctx, client)

	if err != nil {
		t.Fatalf("expected forceHttps with TLS to reconcile, got %v", err)
	}
	if !platformIngressUsesTLS {
		t.Fatalf("expected platform ingress patch to reference TLS secret")
	}
}

// TestReconcilePlatformTLSSecretEventAppliesPlatformIngress verifies TLS Secret watch behavior.
// Given kite/global-tls-secret changes after a baseDomain has already been configured.
// When the Secret event handler runs.
// Then the platform Ingress is reconciled so spec.tls can reference the Secret.
func TestReconcilePlatformTLSSecretEventAppliesPlatformIngress(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), platformIngressReconcileListKinds(),
		newPlatformIngressRuntimeConfig(false),
		newPlatformIngressTLSSecret(),
	)
	appliedPlatformIngress := false
	client.Fake.PrependReactor("patch", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchAction, ok := action.(k8stesting.PatchAction)
		if ok && patchAction.GetName() == "kite-platform" {
			appliedPlatformIngress = true
		}
		return true, newPlatformIngressObject("kite-platform"), nil
	})

	reconcilePlatformTLSSecretEvent(client, newPlatformIngressTLSSecret())

	if _, err := client.Resource(configMapGVR).Namespace(kiteGlobalConfigNamespace).Get(ctx, kiteGlobalConfigName, metav1.GetOptions{}); err != nil {
		t.Fatalf("expected runtime config fixture to remain readable, got %v", err)
	}
	if !appliedPlatformIngress {
		t.Fatalf("expected TLS Secret event to apply platform ingress")
	}
}

// TestReconcilePlatformTLSSecretEventIgnoresOtherSecrets verifies event filtering.
// Given a non-platform Secret changes in the kite namespace.
// When the Secret event handler runs.
// Then the platform Ingress reconcile path is not invoked.
func TestReconcilePlatformTLSSecretEventIgnoresOtherSecrets(t *testing.T) {
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), platformIngressReconcileListKinds(),
		newPlatformIngressRuntimeConfig(false),
	)
	appliedPlatformIngress := false
	client.Fake.PrependReactor("patch", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		appliedPlatformIngress = true
		return true, newPlatformIngressObject("kite-platform"), nil
	})

	other := newPlatformIngressTLSSecret()
	other.SetName("unrelated")
	reconcilePlatformTLSSecretEvent(client, other)

	if appliedPlatformIngress {
		t.Fatalf("expected unrelated Secret event to be ignored")
	}
}

// platformIngressReconcileListKinds returns fake dynamic client list kind mappings.
// The returned map covers runtime ConfigMaps, networking Ingresses, and Traefik middleware.
// This helper is used by platform ingress reconcile unit tests.
func platformIngressReconcileListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		configMapGVR:         "ConfigMapList",
		secretGVR:            "SecretList",
		ingressGVR:           "IngressList",
		traefikMiddlewareGVR: "MiddlewareList",
	}
}

// newPlatformIngressRuntimeConfig creates the kite-runtime-config test object.
// forceHTTPS controls the forceHttps data value used by the platform ingress reconciler.
// The returned ConfigMap is used by tests that exercise HTTP and HTTPS ingress modes.
func newPlatformIngressRuntimeConfig(forceHTTPS bool) *unstructured.Unstructured {
	return newPlatformIngressRuntimeConfigWithBaseDomain("hy3on.site", forceHTTPS)
}

// newPlatformIngressRuntimeConfigWithBaseDomain creates the kite-runtime-config test object.
// baseDomain controls data.baseDomain so tests can exercise configured and unconfigured hosts.
// forceHTTPS controls the forceHttps data value used by the platform ingress reconciler.
func newPlatformIngressRuntimeConfigWithBaseDomain(baseDomain string, forceHTTPS bool) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      kiteGlobalConfigName,
				"namespace": kiteGlobalConfigNamespace,
			},
			"data": map[string]any{
				kiteGlobalBaseDomainKey:    baseDomain,
				config.ForceHTTPSConfigKey: fmt.Sprintf("%t", forceHTTPS),
			},
		},
	}
}

// newPlatformIngressTLSSecret creates a test TLS Secret for platform Ingress reconcile.
// The returned object uses the real platform namespace and name constants.
// Tests use it to verify forceHttps refuses missing certificates and wires present certificates.
func newPlatformIngressTLSSecret() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      platform.GlobalTLSSecretName,
				"namespace": platform.GlobalTLSSecretNS,
			},
			"type": "kubernetes.io/tls",
			"data": map[string]any{
				platform.TLSCertificateKey:    "Y2VydA==",
				platform.TLSPrivateKeyDataKey: "a2V5",
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

// newPlatformHTTPSRedirectMiddlewareObject creates a minimal Traefik Middleware test object.
// The returned object lets fake dynamic clients exercise HTTPS redirect apply calls.
// Tests only need metadata because middleware rendering is covered by its own renderer package.
func newPlatformHTTPSRedirectMiddlewareObject() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "Middleware",
			"metadata": map[string]any{
				"name":      "kite-platform-https-redirect",
				"namespace": kiteGlobalConfigNamespace,
			},
		},
	}
}
