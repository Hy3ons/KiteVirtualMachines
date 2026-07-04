package config

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	KiteNamespace                = "kite"
	RuntimeConfigName            = "kite-runtime-config"
	RuntimeSecretName            = "kite-runtime-secret"
	DefaultAccessTokenTTLMinutes = 60
	JWTSecretKey                 = "jwtSecret"
	PasswordSaltKey              = "passwordSalt"
	AccessTokenTTLMinutesKey     = "accessTokenTTLMinutes"
	VMStorageClassNameKey        = "vmStorageClassName"
	GoldenImageNamespaceKey      = "goldenImageNamespace"
	DefaultVMImageKey            = "defaultVmImage"
	ForceHTTPSConfigKey          = "forceHttps"
	AdminContactKey              = "adminContact"
	DefaultVMStorageClassName    = "kite-vm-storage"
	DefaultGoldenImageNamespace  = "kite"
	DefaultVMImage               = "ubuntu-22.04"
)

var configMapGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "configmaps",
}

var secretGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "secrets",
}

var namespaceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}

// Config is the runtime configuration used by kite-api.
// JWTSecret signs API access tokens.
// PasswordSalt hashes and verifies KiteUser passwords.
// AccessTokenTTL controls issued access token lifetime.
type Config struct {
	JWTSecret      string
	PasswordSalt   string
	AccessTokenTTL time.Duration
}

// Bootstrap loads Kite runtime configuration from a ConfigMap and Secret.
// ctx controls Kubernetes API calls.
// dynamicClient reads and writes kite/kite-runtime-config and kite/kite-runtime-secret.
// Missing secret fields are generated once or migrated from legacy ConfigMap data before Config is returned.
func Bootstrap(ctx context.Context, dynamicClient dynamic.Interface) (Config, error) {
	if dynamicClient == nil {
		return Config{}, fmt.Errorf("dynamic Kubernetes client is required")
	}
	if err := ensureKiteNamespace(ctx, dynamicClient); err != nil {
		return Config{}, err
	}

	configMap, legacySecrets, err := ensureRuntimeConfigMap(ctx, dynamicClient)
	if err != nil {
		return Config{}, err
	}
	runtimeSecret, err := ensureRuntimeSecret(ctx, dynamicClient, legacySecrets)
	if err != nil {
		return Config{}, err
	}
	configMap, err = clearLegacyRuntimeConfigSecrets(ctx, dynamicClient, configMap)
	if err != nil {
		return Config{}, err
	}

	return configFromResources(configMap, runtimeSecret)
}

// Load reads existing Kite runtime configuration without creating or repairing resources.
// ctx controls Kubernetes API calls.
// dynamicClient reads kite/kite-runtime-config and kite/kite-runtime-secret.
// This function is used by components that should not own runtime secret bootstrap.
func Load(ctx context.Context, dynamicClient dynamic.Interface) (Config, error) {
	if dynamicClient == nil {
		return Config{}, fmt.Errorf("dynamic Kubernetes client is required")
	}
	configMap, err := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Get(ctx, RuntimeConfigName, metav1.GetOptions{})
	if err != nil {
		return Config{}, fmt.Errorf("failed to read runtime config: %w", err)
	}
	runtimeSecret, err := dynamicClient.Resource(secretGVR).Namespace(KiteNamespace).Get(ctx, RuntimeSecretName, metav1.GetOptions{})
	if err != nil {
		return Config{}, fmt.Errorf("failed to read runtime secret: %w", err)
	}

	return configFromResources(configMap, runtimeSecret)
}

// ensureRuntimeConfigMap creates or normalizes public runtime settings.
// ctx controls Kubernetes ConfigMap requests.
// dynamicClient reads and writes kite/kite-runtime-config.
// The returned ConfigMap may still include legacy secret keys until Secret migration succeeds.
func ensureRuntimeConfigMap(ctx context.Context, dynamicClient dynamic.Interface) (*unstructured.Unstructured, map[string]string, error) {
	configMap, err := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Get(ctx, RuntimeConfigName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		created, createErr := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Create(ctx, newRuntimeConfigMap(), metav1.CreateOptions{})
		if createErr != nil {
			return nil, nil, fmt.Errorf("failed to create runtime config: %w", createErr)
		}
		return created, map[string]string{}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read runtime config: %w", err)
	}

	legacySecrets := legacySecretData(configMap)
	next := configMap.DeepCopy()
	data, changed := normalizedRuntimeData(next)
	if changed {
		if err := unstructured.SetNestedStringMap(next.Object, data, "data"); err != nil {
			return nil, nil, err
		}
		updated, updateErr := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Update(ctx, next, metav1.UpdateOptions{})
		if updateErr != nil {
			return nil, nil, fmt.Errorf("failed to update runtime config defaults: %w", updateErr)
		}
		return updated, legacySecrets, nil
	}

	return configMap, legacySecrets, nil
}

// clearLegacyRuntimeConfigSecrets finalizes migration from ConfigMap to Secret storage.
// ctx controls the Kubernetes ConfigMap update request.
// dynamicClient updates kite/kite-runtime-config only after kite-runtime-secret is confirmed.
// The returned ConfigMap contains only public runtime settings.
func clearLegacyRuntimeConfigSecrets(ctx context.Context, dynamicClient dynamic.Interface, configMap *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	next := configMap.DeepCopy()
	data, _, _ := unstructured.NestedStringMap(next.Object, "data")
	changed := false
	if _, exists := data[JWTSecretKey]; exists {
		delete(data, JWTSecretKey)
		changed = true
	}
	if _, exists := data[PasswordSaltKey]; exists {
		delete(data, PasswordSaltKey)
		changed = true
	}
	if !changed {
		return configMap, nil
	}
	if err := unstructured.SetNestedStringMap(next.Object, data, "data"); err != nil {
		return nil, err
	}
	updated, err := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Update(ctx, next, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to remove legacy runtime config secrets: %w", err)
	}
	return updated, nil
}

// ensureRuntimeSecret creates or normalizes private runtime secrets.
// ctx controls Kubernetes Secret requests.
// dynamicClient reads and writes kite/kite-runtime-secret.
// legacySecrets provides migration values from older kite-runtime-config data.
func ensureRuntimeSecret(ctx context.Context, dynamicClient dynamic.Interface, legacySecrets map[string]string) (*unstructured.Unstructured, error) {
	runtimeSecret, err := dynamicClient.Resource(secretGVR).Namespace(KiteNamespace).Get(ctx, RuntimeSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		created, createErr := dynamicClient.Resource(secretGVR).Namespace(KiteNamespace).Create(ctx, newRuntimeSecret(legacySecrets), metav1.CreateOptions{})
		if createErr != nil {
			return nil, fmt.Errorf("failed to create runtime secret: %w", createErr)
		}
		return created, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read runtime secret: %w", err)
	}

	next := runtimeSecret.DeepCopy()
	data, changed := normalizedRuntimeSecretData(next, legacySecrets)
	if changed {
		if err := unstructured.SetNestedStringMap(next.Object, data, "stringData"); err != nil {
			return nil, err
		}
		unstructured.RemoveNestedField(next.Object, "data")
		updated, updateErr := dynamicClient.Resource(secretGVR).Namespace(KiteNamespace).Update(ctx, next, metav1.UpdateOptions{})
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update runtime secret defaults: %w", updateErr)
		}
		return updated, nil
	}

	return runtimeSecret, nil
}

// ensureKiteNamespace creates the kite namespace when local development starts from a clean cluster.
// ctx controls Kubernetes API calls.
// dynamicClient writes core/v1 namespaces.
// A nil error means the namespace exists or was created successfully.
func ensureKiteNamespace(ctx context.Context, dynamicClient dynamic.Interface) error {
	namespace := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]any{
				"name": KiteNamespace,
			},
		},
	}

	_, err := dynamicClient.Resource(namespaceGVR).Create(ctx, namespace, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to ensure kite namespace: %w", err)
	}

	return nil
}

// newRuntimeConfigMap creates the initial runtime ConfigMap for kite-api.
// Public runtime defaults are persisted so image users do not need .env files.
// The returned object is written to the kite namespace and intentionally excludes secrets.
func newRuntimeConfigMap() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      RuntimeConfigName,
				"namespace": KiteNamespace,
			},
			"data": stringMapToAny(defaultRuntimeData()),
		},
	}
}

// normalizedRuntimeData fills missing runtime config keys.
// obj is the existing runtime ConfigMap.
// The returned map contains every public key and whether anything changed.
func normalizedRuntimeData(obj *unstructured.Unstructured) (map[string]string, bool) {
	data, _, _ := unstructured.NestedStringMap(obj.Object, "data")
	if data == nil {
		data = map[string]string{}
	}

	changed := false
	defaults := defaultRuntimeData()
	for key, value := range defaults {
		if data[key] == "" {
			data[key] = value
			changed = true
		}
	}

	return data, changed
}

// defaultRuntimeData returns default config data for the first API startup.
// Storage and image defaults are used by production k3s installs with Longhorn.
// Secret material is stored separately in kite-runtime-secret.
func defaultRuntimeData() map[string]string {
	return map[string]string{
		AccessTokenTTLMinutesKey: strconv.Itoa(DefaultAccessTokenTTLMinutes),
		VMStorageClassNameKey:    DefaultVMStorageClassName,
		GoldenImageNamespaceKey:  DefaultGoldenImageNamespace,
		DefaultVMImageKey:        DefaultVMImage,
		ForceHTTPSConfigKey:      strconv.FormatBool(false),
		AdminContactKey:          "",
	}
}

// configFromResources converts runtime ConfigMap and Secret objects into Config.
// configMap contains public settings such as accessTokenTTLMinutes.
// runtimeSecret contains jwtSecret and passwordSalt values.
// The returned Config is validated for required values.
func configFromResources(configMap *unstructured.Unstructured, runtimeSecret *unstructured.Unstructured) (Config, error) {
	data, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
	secretData := runtimeSecretStringData(runtimeSecret)
	jwtSecret := secretData[JWTSecretKey]
	passwordSalt := secretData[PasswordSaltKey]
	if jwtSecret == "" || passwordSalt == "" {
		return Config{}, fmt.Errorf("runtime config is missing required fields")
	}

	ttlMinutes := DefaultAccessTokenTTLMinutes
	if value := data[AccessTokenTTLMinutesKey]; value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return Config{}, fmt.Errorf("accessTokenTTLMinutes must be a positive integer")
		}
		ttlMinutes = parsed
	}

	return Config{
		JWTSecret:      jwtSecret,
		PasswordSalt:   passwordSalt,
		AccessTokenTTL: time.Duration(ttlMinutes) * time.Minute,
	}, nil
}

// GenerateSecret creates a URL-safe random value with a timestamp prefix.
// prefix labels the generated value for easier human inspection.
// The returned string is suitable for JWT signing and password salt storage.
// This function is used by initial config bootstrap and admin secret rotation.
func GenerateSecret(prefix string) string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + base64.RawURLEncoding.EncodeToString(bytes)
}
