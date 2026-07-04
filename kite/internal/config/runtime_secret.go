package config

import (
	"encoding/base64"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newRuntimeSecret creates the initial runtime Secret for kite-api and kite-gateway.
// values may contain legacy jwtSecret and passwordSalt data migrated from the ConfigMap.
// The returned object is written to the kite namespace with stringData for Kubernetes encoding.
func newRuntimeSecret(values map[string]string) *unstructured.Unstructured {
	data := defaultRuntimeSecretData(values)
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      RuntimeSecretName,
				"namespace": KiteNamespace,
			},
			"type":       "Opaque",
			"stringData": stringMapToAny(data),
		},
	}
}

// normalizedRuntimeSecretData fills missing runtime secret keys.
// obj is the existing runtime Secret.
// legacyValues provide migration data from older runtime ConfigMaps.
func normalizedRuntimeSecretData(obj *unstructured.Unstructured, legacyValues map[string]string) (map[string]string, bool) {
	data := runtimeSecretStringData(obj)
	changed := false
	defaults := defaultRuntimeSecretData(legacyValues)
	for key, value := range defaults {
		if data[key] == "" {
			data[key] = value
			changed = true
		}
	}

	return data, changed
}

// defaultRuntimeSecretData returns required runtime secret values.
// values can override generated defaults during legacy ConfigMap migration.
// The returned map always includes JWTSecretKey and PasswordSaltKey.
func defaultRuntimeSecretData(values map[string]string) map[string]string {
	data := map[string]string{
		JWTSecretKey:    GenerateSecret("jwt"),
		PasswordSaltKey: GenerateSecret("salt"),
	}
	if values[JWTSecretKey] != "" {
		data[JWTSecretKey] = values[JWTSecretKey]
	}
	if values[PasswordSaltKey] != "" {
		data[PasswordSaltKey] = values[PasswordSaltKey]
	}

	return data
}

// legacySecretData reads legacy secret fields from a runtime ConfigMap.
// obj is the possibly old kite-runtime-config object.
// The returned map is used once when creating or repairing kite-runtime-secret.
func legacySecretData(obj *unstructured.Unstructured) map[string]string {
	data, _, _ := unstructured.NestedStringMap(obj.Object, "data")
	return map[string]string{
		JWTSecretKey:    data[JWTSecretKey],
		PasswordSaltKey: data[PasswordSaltKey],
	}
}

// stringMapToAny converts string maps into JSON-compatible unstructured object maps.
// values is copied so callers can keep mutating their typed map after object construction.
// The returned map is safe for unstructured.Unstructured DeepCopy.
func stringMapToAny(values map[string]string) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

// runtimeSecretStringData returns decoded Secret values as strings.
// obj is a Kubernetes Secret that may contain stringData or base64-encoded data.
// The returned map is used by Bootstrap and admin settings reads.
func runtimeSecretStringData(obj *unstructured.Unstructured) map[string]string {
	stringData, _, _ := unstructured.NestedStringMap(obj.Object, "stringData")
	if len(stringData) > 0 {
		return stringData
	}

	encodedData, _, _ := unstructured.NestedStringMap(obj.Object, "data")
	data := map[string]string{}
	for key, value := range encodedData {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err == nil {
			data[key] = string(decoded)
		}
	}
	return data
}
