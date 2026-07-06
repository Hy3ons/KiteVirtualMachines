package platform

import (
	"context"
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
	GlobalTLSSecretName     = "global-tls-secret"
	GlobalTLSSecretNS       = config.KiteNamespace
	LegacyGlobalTLSSecretNS = "kube-system"
	BaseDomainConfigKey     = "baseDomain"
	TLSCertificateKey       = "tls.crt"
	TLSPrivateKeyDataKey    = "tls.key"
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
// HasTLSCertificate reports whether kite/global-tls-secret contains TLS material.
// This struct is returned by kite-api config endpoints.
type Settings struct {
	BaseDomain        string                  `json:"baseDomain"`
	ForceHTTPS        bool                    `json:"forceHttps"`
	AdminContact      string                  `json:"adminContact"`
	HasJWTSecret      bool                    `json:"hasJWTSecret"`
	HasPasswordSalt   bool                    `json:"hasPasswordSalt"`
	HasTLSCertificate bool                    `json:"hasTLSCertificate"`
	SSHGateway        SSHGatewayAdminSettings `json:"sshGateway"`
}

// PublicSettings contains frontend-readable settings for unauthenticated and ordinary user pages.
// SSHGateway omits host fallback details because those are operator-only settings.
type PublicSettings struct {
	BaseDomain        string                   `json:"baseDomain"`
	ForceHTTPS        bool                     `json:"forceHttps"`
	AdminContact      string                   `json:"adminContact"`
	HasTLSCertificate bool                     `json:"hasTLSCertificate"`
	SSHGateway        SSHGatewayPublicSettings `json:"sshGateway"`
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

	hasTLS, err := s.HasTLSCertificate(ctx)
	if err != nil {
		return Settings{}, err
	}
	status, err := s.GetSSHGatewayStatus(ctx)
	if err != nil {
		return Settings{}, err
	}
	secretData, err := s.runtimeSecretData(ctx)
	if err != nil {
		return Settings{}, err
	}
	desired := SSHGatewayDesiredFromConfigData(data)

	return Settings{
		BaseDomain:        data[BaseDomainConfigKey],
		ForceHTTPS:        strings.EqualFold(data[config.ForceHTTPSConfigKey], "true"),
		AdminContact:      data[config.AdminContactKey],
		HasJWTSecret:      secretData[config.JWTSecretKey] != "",
		HasPasswordSalt:   secretData[config.PasswordSaltKey] != "",
		HasTLSCertificate: hasTLS,
		SSHGateway:        desired.Admin(status),
	}, nil
}

// GetPublic returns frontend-readable platform settings without operator-only SSH fallback values.
// ctx controls Kubernetes API calls.
// This function is used by the public /api/v1/config route.
func (s *Service) GetPublic(ctx context.Context) (PublicSettings, error) {
	data, err := s.runtimeConfigData(ctx)
	if err != nil {
		return PublicSettings{}, err
	}
	hasTLS, err := s.HasTLSCertificate(ctx)
	if err != nil {
		return PublicSettings{}, err
	}
	status, err := s.GetSSHGatewayStatus(ctx)
	if err != nil {
		return PublicSettings{}, err
	}
	desired := SSHGatewayDesiredFromConfigData(data)

	return PublicSettings{
		BaseDomain:        data[BaseDomainConfigKey],
		ForceHTTPS:        strings.EqualFold(data[config.ForceHTTPSConfigKey], "true"),
		AdminContact:      data[config.AdminContactKey],
		HasTLSCertificate: hasTLS,
		SSHGateway:        desired.Public(status),
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
	if forceHTTPS {
		hasTLS, err := s.HasTLSCertificate(ctx)
		if err != nil {
			return Settings{}, err
		}
		if !hasTLS {
			return Settings{}, fmt.Errorf("TLS certificate must be uploaded before forceHttps can be enabled")
		}
	}

	return s.updateRuntimeConfigValue(ctx, config.ForceHTTPSConfigKey, strconv.FormatBool(forceHTTPS))
}

// UpdateAdminContact stores the operator contact string in kite/kite-runtime-config.
// ctx controls Kubernetes API create or update requests.
// adminContact is a free-form email, phone number, chat handle, or URL shown to users without VM create access.
// The returned Settings value reflects the updated config.
func (s *Service) UpdateAdminContact(ctx context.Context, adminContact string) (Settings, error) {
	return s.updateRuntimeConfigValue(ctx, config.AdminContactKey, strings.TrimSpace(adminContact))
}

// RotateRuntimeSecrets replaces JWT and password salt values in kite/kite-runtime-secret.
// ctx controls Kubernetes API requests.
// rotateJWTSecret controls whether stringData.jwtSecret is regenerated.
// rotatePasswordSalt controls whether stringData.passwordSalt is regenerated.
// The returned Settings value reports the stored runtime config after rotation.
func (s *Service) RotateRuntimeSecrets(ctx context.Context, rotateJWTSecret bool, rotatePasswordSalt bool) (Settings, error) {
	if !rotateJWTSecret && !rotatePasswordSalt {
		return Settings{}, fmt.Errorf("at least one runtime secret must be selected")
	}

	current, err := s.dynamicClient.Resource(secretGVR).Namespace(config.KiteNamespace).Get(ctx, config.RuntimeSecretName, metav1.GetOptions{})
	if err != nil {
		return Settings{}, err
	}

	next := current.DeepCopy()
	data, _, _ := unstructured.NestedStringMap(next.Object, "stringData")
	if len(data) == 0 {
		data = decodedSecretStringData(next)
	}
	if data == nil {
		data = map[string]string{}
	}
	if rotateJWTSecret {
		data[config.JWTSecretKey] = config.GenerateSecret("jwt")
	}
	if rotatePasswordSalt {
		data[config.PasswordSaltKey] = config.GenerateSecret("salt")
	}
	if err := unstructured.SetNestedStringMap(next.Object, data, "stringData"); err != nil {
		return Settings{}, err
	}
	unstructured.RemoveNestedField(next.Object, "data")

	if _, err := s.dynamicClient.Resource(secretGVR).Namespace(config.KiteNamespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return Settings{}, err
	}

	return s.Get(ctx)
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
