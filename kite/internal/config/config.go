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

// Bootstrap loads kite-api runtime configuration from a Kubernetes ConfigMap.
// ctx controls Kubernetes API calls.
// dynamicClient reads and writes kite/kite-runtime-config.
// Missing ConfigMap fields are generated once and persisted before the Config is returned.
func Bootstrap(ctx context.Context, dynamicClient dynamic.Interface) (Config, error) {
	if dynamicClient == nil {
		return Config{}, fmt.Errorf("dynamic Kubernetes client is required")
	}
	if err := ensureKiteNamespace(ctx, dynamicClient); err != nil {
		return Config{}, err
	}

	configMap, err := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Get(ctx, RuntimeConfigName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		created, createErr := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Create(ctx, newRuntimeConfigMap(), metav1.CreateOptions{})
		if createErr != nil {
			return Config{}, fmt.Errorf("failed to create runtime config: %w", createErr)
		}
		return configFromObject(created)
	}
	if err != nil {
		return Config{}, fmt.Errorf("failed to read runtime config: %w", err)
	}

	next := configMap.DeepCopy()
	data, changed := normalizedRuntimeData(next)
	if changed {
		if err := unstructured.SetNestedStringMap(next.Object, data, "data"); err != nil {
			return Config{}, err
		}
		updated, updateErr := dynamicClient.Resource(configMapGVR).Namespace(KiteNamespace).Update(ctx, next, metav1.UpdateOptions{})
		if updateErr != nil {
			return Config{}, fmt.Errorf("failed to update runtime config defaults: %w", updateErr)
		}
		return configFromObject(updated)
	}

	return configFromObject(configMap)
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
// Generated secrets are persisted so image users do not need .env files.
// The returned object is written to the kite namespace.
func newRuntimeConfigMap() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      RuntimeConfigName,
				"namespace": KiteNamespace,
			},
			"data": defaultRuntimeData(),
		},
	}
}

// normalizedRuntimeData fills missing runtime config keys.
// obj is the existing runtime ConfigMap.
// The returned map contains every required key and whether anything changed.
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
// JWTSecret and PasswordSalt are generated once.
// Storage and image defaults are used by production k3s installs with Longhorn.
func defaultRuntimeData() map[string]string {
	return map[string]string{
		JWTSecretKey:             GenerateSecret("jwt"),
		PasswordSaltKey:          GenerateSecret("salt"),
		AccessTokenTTLMinutesKey: strconv.Itoa(DefaultAccessTokenTTLMinutes),
		VMStorageClassNameKey:    DefaultVMStorageClassName,
		GoldenImageNamespaceKey:  DefaultGoldenImageNamespace,
		DefaultVMImageKey:        DefaultVMImage,
		ForceHTTPSConfigKey:      strconv.FormatBool(false),
		AdminContactKey:          "",
	}
}

// configFromObject converts a runtime ConfigMap into Config.
// obj is the unstructured ConfigMap returned by Kubernetes.
// The returned Config is validated for required values.
func configFromObject(obj *unstructured.Unstructured) (Config, error) {
	data, _, _ := unstructured.NestedStringMap(obj.Object, "data")
	jwtSecret := data[JWTSecretKey]
	passwordSalt := data[PasswordSaltKey]
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
