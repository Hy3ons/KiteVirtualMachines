package platform

import (
	"context"
	"encoding/base64"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/config"
)

// runtimeSecretData reads kite/kite-runtime-secret and returns decoded secret data.
// ctx controls the Kubernetes Secret get request.
// The returned map contains jwtSecret and passwordSalt when runtime bootstrap has completed.
func (s *Service) runtimeSecretData(ctx context.Context) (map[string]string, error) {
	secret, err := s.dynamicClient.Resource(secretGVR).Namespace(config.KiteNamespace).Get(ctx, config.RuntimeSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	return decodedSecretStringData(secret), nil
}

// decodedSecretStringData returns Kubernetes Secret values as plain strings.
// secret may contain stringData in fake clients or base64-encoded data in real Kubernetes reads.
// The returned map is used only for settings status and runtime secret rotation.
func decodedSecretStringData(secret *unstructured.Unstructured) map[string]string {
	stringData, _, _ := unstructured.NestedStringMap(secret.Object, "stringData")
	if len(stringData) > 0 {
		return stringData
	}

	encodedData, _, _ := unstructured.NestedStringMap(secret.Object, "data")
	data := map[string]string{}
	for key, value := range encodedData {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err == nil {
			data[key] = string(decoded)
		}
	}
	return data
}
