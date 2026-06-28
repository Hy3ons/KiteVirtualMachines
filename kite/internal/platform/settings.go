package platform

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"kite/internal/config"
)

const (
	GlobalTLSSecretName  = "global-tls-secret"
	GlobalTLSSecretNS    = "kube-system"
	BaseDomainConfigKey  = "baseDomain"
	TLSCertificateKey    = "tls.crt"
	TLSPrivateKeyDataKey = "tls.key"
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

// Settings contains cluster-wide frontend configuration values.
// BaseDomain is used by platform and VM Ingress generation.
// ForceHTTPS reports whether the platform Ingress should redirect HTTP traffic to HTTPS.
// HasTLSCertificate reports whether kube-system/global-tls-secret contains TLS material.
// This struct is returned by kite-api config endpoints.
type Settings struct {
	BaseDomain        string `json:"baseDomain"`
	ForceHTTPS        bool   `json:"forceHttps"`
	HasJWTSecret      bool   `json:"hasJWTSecret"`
	HasPasswordSalt   bool   `json:"hasPasswordSalt"`
	HasTLSCertificate bool   `json:"hasTLSCertificate"`
}

// Service manages Kite platform-wide settings stored in Kubernetes resources.
// dynamicClient reads the Kite ConfigMap and writes TLS Secret resources for the ingress controller.
// This service is used by admin settings API handlers and public config reads.
type Service struct {
	dynamicClient dynamic.Interface
}

// NewService creates a platform settings service.
// dynamicClient is used for Kubernetes ConfigMap and Secret operations.
// The returned service is request-scoped by kite-api handlers.
func NewService(dynamicClient dynamic.Interface) *Service {
	return &Service{dynamicClient: dynamicClient}
}

// Get returns the current platform settings.
// ctx controls Kubernetes API calls.
// Missing runtime ConfigMap resources return an error because kite-api should bootstrap them at startup.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	data, err := s.runtimeConfigData(ctx)
	if err != nil {
		return Settings{}, err
	}

	hasTLS, err := s.hasTLSCertificate(ctx)
	if err != nil {
		return Settings{}, err
	}

	return Settings{
		BaseDomain:        data[BaseDomainConfigKey],
		ForceHTTPS:        strings.EqualFold(data[config.ForceHTTPSConfigKey], "true"),
		HasJWTSecret:      data[config.JWTSecretKey] != "",
		HasPasswordSalt:   data[config.PasswordSaltKey] != "",
		HasTLSCertificate: hasTLS,
	}, nil
}

// GetBaseDomain reads kite/kite-runtime-config data.baseDomain.
// ctx controls the Kubernetes API get request.
// The returned string is empty when the ConfigMap or key does not exist.
func (s *Service) GetBaseDomain(ctx context.Context) (string, error) {
	data, err := s.runtimeConfigData(ctx)
	if err != nil {
		return "", err
	}

	return data[BaseDomainConfigKey], nil
}

// UpdateBaseDomain stores a new base domain in kite/kite-runtime-config.
// ctx controls Kubernetes API create or update requests.
// baseDomain is trimmed and stored without a trailing dot.
// The returned Settings value reflects the updated config.
func (s *Service) UpdateBaseDomain(ctx context.Context, baseDomain string) (Settings, error) {
	baseDomain = strings.Trim(strings.TrimSpace(baseDomain), ".")
	if baseDomain == "" {
		return Settings{}, fmt.Errorf("baseDomain is required")
	}

	return s.updateRuntimeConfigValue(ctx, BaseDomainConfigKey, baseDomain)
}

// UpdateForceHTTPS stores the platform HTTPS redirect policy in kite/kite-runtime-config.
// ctx controls Kubernetes API create or update requests.
// forceHTTPS determines whether the controller renders HTTPS redirect settings for platform Ingress.
// The returned Settings value reflects the updated config.
func (s *Service) UpdateForceHTTPS(ctx context.Context, forceHTTPS bool) (Settings, error) {
	return s.updateRuntimeConfigValue(ctx, config.ForceHTTPSConfigKey, strconv.FormatBool(forceHTTPS))
}

// RotateRuntimeSecrets replaces JWT and password salt values in kite/kite-runtime-config.
// ctx controls Kubernetes API requests.
// rotateJWTSecret controls whether data.jwtSecret is regenerated.
// rotatePasswordSalt controls whether data.passwordSalt is regenerated.
// The returned Settings value reports the stored runtime config after rotation.
func (s *Service) RotateRuntimeSecrets(ctx context.Context, rotateJWTSecret bool, rotatePasswordSalt bool) (Settings, error) {
	if !rotateJWTSecret && !rotatePasswordSalt {
		return Settings{}, fmt.Errorf("at least one runtime secret must be selected")
	}

	current, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Get(ctx, config.RuntimeConfigName, metav1.GetOptions{})
	if err != nil {
		return Settings{}, err
	}

	next := current.DeepCopy()
	data, _, _ := unstructured.NestedStringMap(next.Object, "data")
	if data == nil {
		data = map[string]string{}
	}
	if rotateJWTSecret {
		data[config.JWTSecretKey] = config.GenerateSecret("jwt")
	}
	if rotatePasswordSalt {
		data[config.PasswordSaltKey] = config.GenerateSecret("salt")
	}
	if err := unstructured.SetNestedStringMap(next.Object, data, "data"); err != nil {
		return Settings{}, err
	}

	if _, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return Settings{}, err
	}

	return s.Get(ctx)
}

// UpdateTLSCertificate stores wildcard TLS material in kube-system/global-tls-secret.
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

// hasTLSCertificate checks whether kube-system/global-tls-secret contains both TLS data keys.
// ctx controls the Kubernetes API get request.
// The returned boolean is false when the Secret is missing or incomplete.
func (s *Service) hasTLSCertificate(ctx context.Context) (bool, error) {
	secret, err := s.dynamicClient.Resource(secretGVR).Namespace(GlobalTLSSecretNS).Get(ctx, GlobalTLSSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	data, _, _ := unstructured.NestedStringMap(secret.Object, "data")
	return data[TLSCertificateKey] != "" && data[TLSPrivateKeyDataKey] != "", nil
}

// updateRuntimeConfigValue updates one key in kite/kite-runtime-config.
// ctx controls Kubernetes API calls.
// key and value identify the data entry to update.
// The returned Settings value reflects the updated ConfigMap.
func (s *Service) updateRuntimeConfigValue(ctx context.Context, key string, value string) (Settings, error) {
	current, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Get(ctx, config.RuntimeConfigName, metav1.GetOptions{})
	if err != nil {
		return Settings{}, err
	}

	next := current.DeepCopy()
	data, _, _ := unstructured.NestedStringMap(next.Object, "data")
	if data == nil {
		data = map[string]string{}
	}
	data[key] = value
	if err := unstructured.SetNestedStringMap(next.Object, data, "data"); err != nil {
		return Settings{}, err
	}

	if _, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return Settings{}, err
	}

	return s.Get(ctx)
}

// runtimeConfigData reads kite/kite-runtime-config and returns its data map.
// ctx controls the Kubernetes API get request.
// The returned error is non-nil when the runtime ConfigMap is missing or malformed.
func (s *Service) runtimeConfigData(ctx context.Context) (map[string]string, error) {
	configMap, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Get(ctx, config.RuntimeConfigName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("runtime config was not found")
	}
	if err != nil {
		return nil, err
	}

	data, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
	if data == nil {
		return nil, fmt.Errorf("runtime config data is empty")
	}

	return data, nil
}

// newGlobalTLSSecret creates the Kubernetes TLS Secret object for global Ingress TLS.
// tlsCert and tlsKey are PEM strings encoded into Secret data fields.
// The returned object is written in kube-system so cluster ingress integrations can reuse it.
func newGlobalTLSSecret(tlsCert string, tlsKey string) *unstructured.Unstructured {
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
				TLSCertificateKey:    base64.StdEncoding.EncodeToString([]byte(tlsCert)),
				TLSPrivateKeyDataKey: base64.StdEncoding.EncodeToString([]byte(tlsKey)),
			},
		},
	}
}
