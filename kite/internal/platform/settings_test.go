package platform

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/config"
)

func TestSettingsReportsForceHTTPS(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "true"))

	settings, err := NewService(dynamicClient).Get(context.Background())
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	if !settings.ForceHTTPS {
		t.Fatal("expected forceHttps to be true")
	}
}

func TestUpdateForceHTTPSStoresRuntimeConfig(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "false"))

	settings, err := NewService(dynamicClient).UpdateForceHTTPS(context.Background(), true)
	if err != nil {
		t.Fatalf("failed to update HTTPS setting: %v", err)
	}

	if !settings.ForceHTTPS {
		t.Fatal("expected returned settings to enable forceHttps")
	}
}

func TestUpdateAdminContactStoresRuntimeConfig(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "false"))

	settings, err := NewService(dynamicClient).UpdateAdminContact(context.Background(), "ops@example.com")
	if err != nil {
		t.Fatalf("failed to update admin contact: %v", err)
	}

	if settings.AdminContact != "ops@example.com" {
		t.Fatalf("expected admin contact to be stored, got %q", settings.AdminContact)
	}
}

func newPlatformSettingsRuntimeConfig(baseDomain string, forceHTTPS string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      config.RuntimeConfigName,
				"namespace": config.KiteNamespace,
			},
			"data": map[string]any{
				BaseDomainConfigKey:        baseDomain,
				config.ForceHTTPSConfigKey: forceHTTPS,
				config.AdminContactKey:     "",
				config.JWTSecretKey:        "jwt",
				config.PasswordSaltKey:     "salt",
			},
		},
	}
}
