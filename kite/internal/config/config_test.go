package config

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

// TestBootstrapCreatesRuntimeSecretSeparately verifies first-start secret storage.
// Given a clean cluster with no Kite runtime resources.
// When Bootstrap initializes runtime config.
// Then jwtSecret and passwordSalt are stored in a Secret, not the public ConfigMap.
func TestBootstrapCreatesRuntimeSecretSeparately(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), configTestListKinds())

	cfg, err := Bootstrap(ctx, client)
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	if cfg.JWTSecret == "" || cfg.PasswordSalt == "" {
		t.Fatalf("expected generated runtime secrets, got %#v", cfg)
	}
	configMap := getConfigTestRuntimeConfigMap(t, ctx, client)
	configData, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
	if configData[JWTSecretKey] != "" || configData[PasswordSaltKey] != "" {
		t.Fatalf("expected runtime ConfigMap to exclude secrets, got %#v", configData)
	}
	secret := getConfigTestRuntimeSecret(t, ctx, client)
	secretData := runtimeSecretStringData(secret)
	if secretData[JWTSecretKey] != cfg.JWTSecret || secretData[PasswordSaltKey] != cfg.PasswordSalt {
		t.Fatalf("expected runtime Secret data to match returned config, got %#v", secretData)
	}
}

// TestBootstrapMigratesLegacyConfigMapSecrets verifies backward-compatible migration.
// Given an older kite-runtime-config stores jwtSecret and passwordSalt directly.
// When Bootstrap runs after the upgrade.
// Then the legacy values move to kite-runtime-secret and are removed from the ConfigMap.
func TestBootstrapMigratesLegacyConfigMapSecrets(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), configTestListKinds(), newConfigTestLegacyRuntimeConfigMap())

	cfg, err := Bootstrap(ctx, client)
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	if cfg.JWTSecret != "legacy-jwt" || cfg.PasswordSalt != "legacy-salt" {
		t.Fatalf("expected legacy secrets to be preserved, got %#v", cfg)
	}
	configMap := getConfigTestRuntimeConfigMap(t, ctx, client)
	configData, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
	if configData[JWTSecretKey] != "" || configData[PasswordSaltKey] != "" {
		t.Fatalf("expected legacy secrets to be removed from ConfigMap, got %#v", configData)
	}
	secret := getConfigTestRuntimeSecret(t, ctx, client)
	secretData := runtimeSecretStringData(secret)
	if secretData[JWTSecretKey] != "legacy-jwt" || secretData[PasswordSaltKey] != "legacy-salt" {
		t.Fatalf("expected legacy secrets in runtime Secret, got %#v", secretData)
	}
}

// TestBootstrapKeepsLegacyConfigMapSecretsWhenSecretCreateFails verifies two-phase migration safety.
// Given legacy runtime secrets exist only in kite-runtime-config.
// When creating kite-runtime-secret fails.
// Then Bootstrap returns an error and leaves legacy ConfigMap secret keys untouched.
func TestBootstrapKeepsLegacyConfigMapSecretsWhenSecretCreateFails(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), configTestListKinds(), newConfigTestLegacyRuntimeConfigMap())
	client.Fake.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, assertiveTestError("secret create failed")
	})

	cfg, err := Bootstrap(ctx, client)
	if err == nil {
		t.Fatalf("expected Secret create failure, got config %#v", cfg)
	}

	configMap := getConfigTestRuntimeConfigMap(t, ctx, client)
	configData, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
	if configData[JWTSecretKey] != "legacy-jwt" || configData[PasswordSaltKey] != "legacy-salt" {
		t.Fatalf("expected legacy ConfigMap secrets to remain after failed migration, got %#v", configData)
	}
}

type assertiveTestError string

func (e assertiveTestError) Error() string {
	return string(e)
}

// configTestListKinds returns fake dynamic client list kind mappings for runtime bootstrap tests.
// The returned map covers namespaces, ConfigMaps, and Secrets used by Bootstrap.
func configTestListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		namespaceGVR: "NamespaceList",
		configMapGVR: "ConfigMapList",
		secretGVR:    "SecretList",
	}
}

// getConfigTestRuntimeConfigMap reads the runtime ConfigMap fixture from the fake client.
// t is the Go test handle used for assertion failures.
// The returned object lets tests inspect migrated public settings.
func getConfigTestRuntimeConfigMap(t *testing.T, ctx context.Context, client *fake.FakeDynamicClient) *unstructured.Unstructured {
	t.Helper()
	configMap, err := client.Resource(configMapGVR).Namespace(KiteNamespace).Get(ctx, RuntimeConfigName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read runtime ConfigMap: %v", err)
	}
	return configMap
}

// getConfigTestRuntimeSecret reads the runtime Secret fixture from the fake client.
// t is the Go test handle used for assertion failures.
// The returned object lets tests inspect generated or migrated secret data.
func getConfigTestRuntimeSecret(t *testing.T, ctx context.Context, client *fake.FakeDynamicClient) *unstructured.Unstructured {
	t.Helper()
	secret, err := client.Resource(secretGVR).Namespace(KiteNamespace).Get(ctx, RuntimeSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read runtime Secret: %v", err)
	}
	return secret
}

// newConfigTestLegacyRuntimeConfigMap creates a pre-Secret runtime ConfigMap fixture.
// The returned object models clusters that stored jwtSecret and passwordSalt before this migration.
func newConfigTestLegacyRuntimeConfigMap() *unstructured.Unstructured {
	data := defaultRuntimeData()
	data[JWTSecretKey] = "legacy-jwt"
	data[PasswordSaltKey] = "legacy-salt"
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      RuntimeConfigName,
				"namespace": KiteNamespace,
			},
			"data": stringMapToAny(data),
		},
	}
}
