package platform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/config"
)

const (
	SSHGatewayStatusConfigName = "kite-gateway-status"

	SSHGatewayPhaseDisabled    = "Disabled"
	SSHGatewayPhaseReconciling = "Reconciling"
	SSHGatewayPhaseReady       = "Ready"
	SSHGatewayPhaseBlocked     = "Blocked"
	SSHGatewayPhaseFailed      = "Failed"

	SSHGatewayReasonExternalDisabled        = "ExternalDisabled"
	SSHGatewayReasonMissingExternalPort     = "MissingExternalPort"
	SSHGatewayReasonMissingHostFallbackPort = "MissingHostFallbackPort"
	SSHGatewayReasonPortConflict            = "PortConflict"
	SSHGatewayReasonServicePending          = "ServicePending"
	SSHGatewayReasonServiceApplied          = "ServiceApplied"
	SSHGatewayReasonApplyFailed             = "ApplyFailed"
)

// SSHGatewayDesired contains operator-owned SSH gateway exposure settings used by the admin API and controller.
type SSHGatewayDesired struct {
	ExternalEnabled     bool   `json:"externalEnabled"`
	ExternalPort        string `json:"externalPort"`
	HostFallbackEnabled bool   `json:"hostFallbackEnabled,omitempty"`
	HostSshdPort        string `json:"hostSshdPort,omitempty"`
}

// SSHGatewayStatus describes the controller-observed SSH gateway exposure state shown in Admin Settings.
type SSHGatewayStatus struct {
	Phase                       string `json:"phase"`
	Reason                      string `json:"reason"`
	Message                     string `json:"message"`
	ObservedExternalPort        string `json:"observedExternalPort,omitempty"`
	ObservedHostFallbackAddress string `json:"observedHostFallbackAddress,omitempty"`
	ObservedServiceName         string `json:"observedServiceName,omitempty"`
	LastTransitionTime          string `json:"lastTransitionTime,omitempty"`
}

// SSHGatewayAdminSettings combines desired and observed gateway state for Level 3 admins.
type SSHGatewayAdminSettings struct {
	ExternalEnabled     bool             `json:"externalEnabled"`
	ExternalPort        string           `json:"externalPort"`
	HostFallbackEnabled bool             `json:"hostFallbackEnabled"`
	HostSshdPort        string           `json:"hostSshdPort"`
	Status              SSHGatewayStatus `json:"status"`
}

// SSHGatewayPublicSettings exposes only user-facing VM SSH connection state and omits host fallback details.
type SSHGatewayPublicSettings struct {
	ExternalEnabled bool   `json:"externalEnabled"`
	ExternalPort    string `json:"externalPort"`
	Phase           string `json:"phase"`
	Reason          string `json:"reason"`
	Message         string `json:"message"`
}

// SSHGatewayDesiredFromConfigData parses SSH gateway desired state from kite-runtime-config data.
func SSHGatewayDesiredFromConfigData(data map[string]string) SSHGatewayDesired {
	return SSHGatewayDesired{
		ExternalEnabled:     strings.EqualFold(data[config.SSHGatewayExternalEnabledKey], "true"),
		ExternalPort:        strings.TrimSpace(data[config.SSHGatewayExternalPortKey]),
		HostFallbackEnabled: strings.EqualFold(data[config.SSHGatewayHostFallbackKey], "true"),
		HostSshdPort:        strings.TrimSpace(data[config.SSHGatewayHostSshdPortKey]),
	}
}

// Public returns the user-safe subset of desired and observed SSH gateway state.
func (d SSHGatewayDesired) Public(status SSHGatewayStatus) SSHGatewayPublicSettings {
	return SSHGatewayPublicSettings{
		ExternalEnabled: d.ExternalEnabled,
		ExternalPort:    d.ExternalPort,
		Phase:           status.Phase,
		Reason:          status.Reason,
		Message:         status.Message,
	}
}

// Admin returns the full Level 3 admin SSH gateway settings payload.
func (d SSHGatewayDesired) Admin(status SSHGatewayStatus) SSHGatewayAdminSettings {
	return SSHGatewayAdminSettings{
		ExternalEnabled:     d.ExternalEnabled,
		ExternalPort:        d.ExternalPort,
		HostFallbackEnabled: d.HostFallbackEnabled,
		HostSshdPort:        d.HostSshdPort,
		Status:              status,
	}
}

// GetSSHGatewayStatus reads kite-gateway-status or returns a safe disabled default before first reconcile.
func (s *Service) GetSSHGatewayStatus(ctx context.Context) (SSHGatewayStatus, error) {
	statusMap, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Get(ctx, SSHGatewayStatusConfigName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return SSHGatewayStatus{
			Phase:   SSHGatewayPhaseDisabled,
			Reason:  SSHGatewayReasonExternalDisabled,
			Message: "SSH gateway external access has not been enabled.",
		}, nil
	}
	if err != nil {
		return SSHGatewayStatus{}, err
	}
	data, _, _ := unstructured.NestedStringMap(statusMap.Object, "data")
	if data == nil {
		data = map[string]string{}
	}
	return SSHGatewayStatusFromData(data), nil
}

// WriteSSHGatewayStatus stores controller-observed gateway state in kite-gateway-status.
func (s *Service) WriteSSHGatewayStatus(ctx context.Context, status SSHGatewayStatus) error {
	if status.LastTransitionTime == "" {
		status.LastTransitionTime = time.Now().UTC().Format(time.RFC3339)
	}
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      SSHGatewayStatusConfigName,
				"namespace": config.KiteNamespace,
			},
			"data": stringMapToAny(SSHGatewayStatusData(status)),
		},
	}
	current, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Get(ctx, SSHGatewayStatusConfigName, metav1.GetOptions{})
	if err == nil {
		obj.SetResourceVersion(current.GetResourceVersion())
		_, err = s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Update(ctx, obj, metav1.UpdateOptions{})
		return err
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	_, err = s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Create(ctx, obj, metav1.CreateOptions{})
	return err
}

// UpdateSSHGateway validates and stores operator desired SSH gateway settings in kite-runtime-config.
func (s *Service) UpdateSSHGateway(ctx context.Context, desired SSHGatewayDesired) (Settings, error) {
	normalized, err := NormalizeSSHGatewayDesired(desired)
	if err != nil {
		return Settings{}, err
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
	data[config.SSHGatewayExternalEnabledKey] = strconv.FormatBool(normalized.ExternalEnabled)
	data[config.SSHGatewayExternalPortKey] = normalized.ExternalPort
	data[config.SSHGatewayHostFallbackKey] = strconv.FormatBool(normalized.HostFallbackEnabled)
	data[config.SSHGatewayHostSshdPortKey] = normalized.HostSshdPort
	if err := unstructured.SetNestedStringMap(next.Object, data, "data"); err != nil {
		return Settings{}, err
	}
	if _, err := s.dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return Settings{}, err
	}
	return s.Get(ctx)
}

// NormalizeSSHGatewayDesired trims and validates operator SSH gateway settings.
func NormalizeSSHGatewayDesired(desired SSHGatewayDesired) (SSHGatewayDesired, error) {
	desired.ExternalPort = strings.TrimSpace(desired.ExternalPort)
	desired.HostSshdPort = strings.TrimSpace(desired.HostSshdPort)
	if desired.ExternalEnabled && !validTCPPort(desired.ExternalPort) {
		return SSHGatewayDesired{}, fmt.Errorf("sshGatewayExternalPort must be a TCP port between 1 and 65535")
	}
	if desired.HostFallbackEnabled && !validTCPPort(desired.HostSshdPort) {
		return SSHGatewayDesired{}, fmt.Errorf("sshGatewayHostSshdPort must be a TCP port between 1 and 65535")
	}
	if desired.ExternalEnabled && desired.HostFallbackEnabled && desired.ExternalPort == desired.HostSshdPort {
		return SSHGatewayDesired{}, fmt.Errorf("sshGatewayExternalPort and sshGatewayHostSshdPort must be different")
	}
	return desired, nil
}

// SSHGatewayStatusFromData converts kite-gateway-status ConfigMap data into a typed payload.
func SSHGatewayStatusFromData(data map[string]string) SSHGatewayStatus {
	phase := data["phase"]
	if phase == "" {
		phase = SSHGatewayPhaseDisabled
	}
	reason := data["reason"]
	if reason == "" {
		reason = SSHGatewayReasonExternalDisabled
	}
	return SSHGatewayStatus{
		Phase:                       phase,
		Reason:                      reason,
		Message:                     data["message"],
		ObservedExternalPort:        data["observedExternalPort"],
		ObservedHostFallbackAddress: data["observedHostFallbackAddress"],
		ObservedServiceName:         data["observedServiceName"],
		LastTransitionTime:          data["lastTransitionTime"],
	}
}

// SSHGatewayStatusData converts a typed status payload into ConfigMap data.
func SSHGatewayStatusData(status SSHGatewayStatus) map[string]string {
	return map[string]string{
		"phase":                       status.Phase,
		"reason":                      status.Reason,
		"message":                     status.Message,
		"observedExternalPort":        status.ObservedExternalPort,
		"observedHostFallbackAddress": status.ObservedHostFallbackAddress,
		"observedServiceName":         status.ObservedServiceName,
		"lastTransitionTime":          status.LastTransitionTime,
	}
}

func validTCPPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port >= 1 && port <= 65535
}

func stringMapToAny(data map[string]string) map[string]any {
	out := make(map[string]any, len(data))
	for key, value := range data {
		out[key] = value
	}
	return out
}
