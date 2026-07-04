package platform

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/config"
)

func TestSettingsReportsForceHTTPS(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
		secretGVR:    "SecretList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "true"), newPlatformRuntimeSecret("jwt", "salt"))

	settings, err := NewService(dynamicClient).Get(context.Background())
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	if !settings.ForceHTTPS {
		t.Fatal("expected forceHttps to be true")
	}
	if !settings.HasJWTSecret || !settings.HasPasswordSalt {
		t.Fatalf("expected runtime Secret flags to be true, got %#v", settings)
	}
}

func TestUpdateForceHTTPSStoresRuntimeConfig(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
		secretGVR:    "SecretList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "false"), newPlatformRuntimeSecret("jwt", "salt"), newPlatformTLSSecret())

	settings, err := NewService(dynamicClient).UpdateForceHTTPS(context.Background(), true)
	if err != nil {
		t.Fatalf("failed to update HTTPS setting: %v", err)
	}

	if !settings.ForceHTTPS {
		t.Fatal("expected returned settings to enable forceHttps")
	}
}

func TestUpdateForceHTTPSRejectsMissingTLSCertificate(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
		secretGVR:    "SecretList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "false"), newPlatformRuntimeSecret("jwt", "salt"))

	settings, err := NewService(dynamicClient).UpdateForceHTTPS(context.Background(), true)
	if err == nil {
		t.Fatalf("expected missing TLS certificate to fail, got %#v", settings)
	}
}

func TestSettingsMigratesLegacyTLSCertificate(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
		secretGVR:    "SecretList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "false"), newPlatformRuntimeSecret("jwt", "salt"), newLegacyPlatformTLSSecret())

	settings, err := NewService(dynamicClient).Get(context.Background())
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	if !settings.HasTLSCertificate {
		t.Fatalf("expected legacy TLS Secret to be reported as migrated, got %#v", settings)
	}
	if _, err := dynamicClient.Resource(secretGVR).Namespace(GlobalTLSSecretNS).Get(context.Background(), GlobalTLSSecretName, metav1.GetOptions{}); err != nil {
		t.Fatalf("expected TLS Secret to be copied into kite namespace, got %v", err)
	}
}

func TestUpdateAdminContactStoresRuntimeConfig(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
		secretGVR:    "SecretList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "false"), newPlatformRuntimeSecret("jwt", "salt"))

	settings, err := NewService(dynamicClient).UpdateAdminContact(context.Background(), "ops@example.com")
	if err != nil {
		t.Fatalf("failed to update admin contact: %v", err)
	}

	if settings.AdminContact != "ops@example.com" {
		t.Fatalf("expected admin contact to be stored, got %q", settings.AdminContact)
	}
}

func TestRotateRuntimeSecretsUpdatesSecret(t *testing.T) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapGVR: "ConfigMapList",
		secretGVR:    "SecretList",
	}, newPlatformSettingsRuntimeConfig("apps.example.com", "false"), newPlatformRuntimeSecret("old-jwt", "old-salt"))

	settings, err := NewService(dynamicClient).RotateRuntimeSecrets(context.Background(), true, true)
	if err != nil {
		t.Fatalf("failed to rotate runtime secrets: %v", err)
	}

	if !settings.HasJWTSecret || !settings.HasPasswordSalt {
		t.Fatalf("expected runtime Secret flags to stay true, got %#v", settings)
	}
	secret, err := dynamicClient.Resource(secretGVR).Namespace(config.KiteNamespace).Get(context.Background(), config.RuntimeSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read rotated runtime Secret: %v", err)
	}
	data := decodedSecretStringData(secret)
	if data[config.JWTSecretKey] == "old-jwt" || data[config.PasswordSaltKey] == "old-salt" {
		t.Fatalf("expected runtime Secret values to rotate, got %#v", data)
	}
}

func newPlatformRuntimeSecret(jwtSecret string, passwordSalt string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      config.RuntimeSecretName,
				"namespace": config.KiteNamespace,
			},
			"type": "Opaque",
			"stringData": map[string]any{
				config.JWTSecretKey:    jwtSecret,
				config.PasswordSaltKey: passwordSalt,
			},
		},
	}
}

func newPlatformTLSSecret() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      GlobalTLSSecretName,
				"namespace": GlobalTLSSecretNS,
			},
			"type": "kubernetes.io/tls",
			"data": map[string]any{
				TLSCertificateKey:    "Y2VydA==",
				TLSPrivateKeyDataKey: "a2V5",
			},
		},
	}
}

func newLegacyPlatformTLSSecret() *unstructured.Unstructured {
	secret := newPlatformTLSSecret()
	secret.SetNamespace(LegacyGlobalTLSSecretNS)
	return secret
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
