package platform

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// UpdateTLSCertificate stores wildcard TLS material in kite/global-tls-secret.
// ctx controls Kubernetes API create or update requests.
// tlsCert and tlsKey are PEM strings from the admin settings page.
// The returned Settings value reports whether TLS material exists after the update.
func (s *Service) UpdateTLSCertificate(ctx context.Context, tlsCert string, tlsKey string) (Settings, error) {
	tlsCert = strings.TrimSpace(tlsCert)
	tlsKey = strings.TrimSpace(tlsKey)
	if tlsCert == "" || tlsKey == "" {
		return Settings{}, fmt.Errorf("tlsCert and tlsKey are required")
	}

	secret := newGlobalTLSSecret(tlsCert, tlsKey)
	current, err := s.dynamicClient.Resource(secretGVR).Namespace(GlobalTLSSecretNS).Get(ctx, GlobalTLSSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, createErr := s.dynamicClient.Resource(secretGVR).Namespace(GlobalTLSSecretNS).Create(ctx, secret, metav1.CreateOptions{}); createErr != nil {
			return Settings{}, createErr
		}
		return s.Get(ctx)
	}
	if err != nil {
		return Settings{}, err
	}

	secret.SetResourceVersion(current.GetResourceVersion())
	if _, err := s.dynamicClient.Resource(secretGVR).Namespace(GlobalTLSSecretNS).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return Settings{}, err
	}

	return s.Get(ctx)
}

// HasTLSCertificate checks whether kite/global-tls-secret contains both TLS data keys.
// ctx controls the Kubernetes API get request.
// The returned boolean is false when the Secret is missing or incomplete.
func (s *Service) HasTLSCertificate(ctx context.Context) (bool, error) {
	secret, err := s.dynamicClient.Resource(secretGVR).Namespace(GlobalTLSSecretNS).Get(ctx, GlobalTLSSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return s.migrateLegacyTLSCertificate(ctx)
	}
	if err != nil {
		return false, err
	}

	data, _, _ := unstructured.NestedStringMap(secret.Object, "data")
	return data[TLSCertificateKey] != "" && data[TLSPrivateKeyDataKey] != "", nil
}

// migrateLegacyTLSCertificate copies an older kube-system TLS Secret into the kite namespace.
// ctx controls Kubernetes Secret read and create requests.
// The returned boolean is true when usable TLS data exists after migration.
func (s *Service) migrateLegacyTLSCertificate(ctx context.Context) (bool, error) {
	legacy, err := s.dynamicClient.Resource(secretGVR).Namespace(LegacyGlobalTLSSecretNS).Get(ctx, GlobalTLSSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	data, _, _ := unstructured.NestedStringMap(legacy.Object, "data")
	if data[TLSCertificateKey] == "" || data[TLSPrivateKeyDataKey] == "" {
		return false, nil
	}
	_, err = s.dynamicClient.Resource(secretGVR).Namespace(GlobalTLSSecretNS).Create(ctx, newGlobalTLSSecretFromEncodedData(data), metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// newGlobalTLSSecret creates the Kubernetes TLS Secret object for global Ingress TLS.
// tlsCert and tlsKey are PEM strings encoded into Secret data fields.
// The returned object is written in the kite namespace because Ingress TLS references are namespace-local.
func newGlobalTLSSecret(tlsCert string, tlsKey string) *unstructured.Unstructured {
	return newGlobalTLSSecretFromEncodedData(map[string]string{
		TLSCertificateKey:    base64.StdEncoding.EncodeToString([]byte(tlsCert)),
		TLSPrivateKeyDataKey: base64.StdEncoding.EncodeToString([]byte(tlsKey)),
	})
}

// newGlobalTLSSecretFromEncodedData creates the namespace-local TLS Secret object from Kubernetes data values.
// data contains base64-encoded tls.crt and tls.key entries.
// The returned object is used by new uploads and legacy kube-system migration.
func newGlobalTLSSecretFromEncodedData(data map[string]string) *unstructured.Unstructured {
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
				TLSCertificateKey:    data[TLSCertificateKey],
				TLSPrivateKeyDataKey: data[TLSPrivateKeyDataKey],
			},
		},
	}
}
