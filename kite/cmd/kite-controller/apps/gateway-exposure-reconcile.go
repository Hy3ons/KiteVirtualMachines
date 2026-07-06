package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"

	"kite/internal/config"
	"kite/internal/kube"
	"kite/internal/platform"
)

const (
	kiteGatewayApplyManager        = "kite-controller-gateway-exposure"
	kiteGatewayInternalServiceName = "kite-gateway"
	kiteGatewayExternalServiceName = "kite-gateway-external"
	kiteGatewayDeploymentName      = "kite-gateway"
	kiteGatewayContainerName       = "kite-gateway"
)

var deploymentGVR = schema.GroupVersionResource{
	Group:    "apps",
	Version:  "v1",
	Resource: "deployments",
}

// RunKiteGatewayExposureReconciler watches runtime config and reconciles SSH gateway exposure.
// clientManager provides the dynamic Kubernetes client.
// stopCh stops the informer during controller shutdown.
// This reconciler owns only kite-gateway-external and gateway fallback env values.
func RunKiteGatewayExposureReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("Kite gateway exposure reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		clientManager.DynamicClient,
		time.Second*30,
		config.KiteNamespace,
		nil,
	)
	configMapInformer := factory.ForResource(configMapGVR).Informer()
	serviceInformer := factory.ForResource(serviceGVR).Informer()
	RegisterKiteGatewayExposureReconciler(configMapInformer, clientManager.DynamicClient)
	RegisterKiteGatewayExposureServiceReconciler(serviceInformer, clientManager.DynamicClient)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, configMapInformer.HasSynced, serviceInformer.HasSynced) {
		log.Printf("failed to sync Kite gateway exposure informer cache")
		return
	}

	if err := ReconcileKiteGatewayExposureFromConfigMap(context.Background(), clientManager.DynamicClient); err != nil {
		log.Printf("failed to reconcile initial Kite gateway exposure: %v", err)
	}

	<-stopCh
}

// RegisterKiteGatewayExposureReconciler attaches runtime ConfigMap event handlers.
// informer watches ConfigMaps in the kite namespace.
// dynamicClient applies gateway Service and Deployment updates.
func RegisterKiteGatewayExposureReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			reconcileGatewayExposureConfigEvent(dynamicClient, obj)
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			reconcileGatewayExposureConfigEvent(dynamicClient, newObj)
		},
	})
}

// RegisterKiteGatewayExposureServiceReconciler attaches external gateway Service event handlers.
// informer watches Services in the kite namespace.
// dynamicClient reads desired config again so status follows LoadBalancer readiness changes.
func RegisterKiteGatewayExposureServiceReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			reconcileGatewayExposureServiceEvent(dynamicClient, obj)
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			reconcileGatewayExposureServiceEvent(dynamicClient, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			reconcileGatewayExposureServiceEvent(dynamicClient, obj)
		},
	})
}

// ReconcileKiteGatewayExposureFromConfigMap reconciles gateway exposure from kite-runtime-config.
// ctx controls Kubernetes API requests.
// dynamicClient reads desired config, writes status, patches gateway fallback env, and applies the external Service.
func ReconcileKiteGatewayExposureFromConfigMap(ctx context.Context, dynamicClient dynamic.Interface) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is required for Kite gateway exposure reconcile")
	}

	configMap, err := dynamicClient.Resource(configMapGVR).Namespace(config.KiteNamespace).Get(ctx, config.RuntimeConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	data, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
	desired := platform.SSHGatewayDesiredFromConfigData(data)
	statusWriter := platform.NewService(dynamicClient)

	if err := normalizeInternalGatewayService(ctx, dynamicClient); err != nil {
		return writeGatewayFailedStatus(ctx, statusWriter, err)
	}

	status, blocked := gatewayBlockedStatus(desired)
	if blocked {
		if err := deleteExternalGatewayService(ctx, dynamicClient); err != nil {
			return writeGatewayFailedStatus(ctx, statusWriter, err)
		}
		if err := patchGatewayFallback(ctx, dynamicClient, false, ""); err != nil {
			return writeGatewayFailedStatus(ctx, statusWriter, err)
		}
		return statusWriter.WriteSSHGatewayStatus(ctx, withTransitionTime(status))
	}

	if !desired.ExternalEnabled {
		if err := deleteExternalGatewayService(ctx, dynamicClient); err != nil {
			return writeGatewayFailedStatus(ctx, statusWriter, err)
		}
		if err := patchGatewayFallback(ctx, dynamicClient, false, ""); err != nil {
			return writeGatewayFailedStatus(ctx, statusWriter, err)
		}
		return statusWriter.WriteSSHGatewayStatus(ctx, withTransitionTime(platform.SSHGatewayStatus{
			Phase:   platform.SSHGatewayPhaseDisabled,
			Reason:  platform.SSHGatewayReasonExternalDisabled,
			Message: "External VM SSH gateway is disabled.",
		}))
	}

	fallbackAddress := ""
	if desired.HostFallbackEnabled {
		fallbackAddress = "$(KITE_NODE_IP):" + desired.HostSshdPort
	}
	if err := patchGatewayFallback(ctx, dynamicClient, desired.HostFallbackEnabled, fallbackAddress); err != nil {
		return writeGatewayFailedStatus(ctx, statusWriter, err)
	}
	if err := applyExternalGatewayService(ctx, dynamicClient, desired.ExternalPort); err != nil {
		return writeGatewayFailedStatus(ctx, statusWriter, err)
	}

	appliedStatus, err := externalGatewayServiceStatus(ctx, dynamicClient, desired.ExternalPort, fallbackAddress)
	if err != nil {
		return writeGatewayFailedStatus(ctx, statusWriter, err)
	}
	return statusWriter.WriteSSHGatewayStatus(ctx, withTransitionTime(appliedStatus))
}

// reconcileGatewayExposureConfigEvent reconciles only kite-runtime-config changes.
// dynamicClient applies the gateway exposure resources.
// obj is the informer event object.
func reconcileGatewayExposureConfigEvent(dynamicClient dynamic.Interface, obj interface{}) {
	resource, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Printf("Kite gateway exposure event is not an unstructured ConfigMap")
		return
	}
	if resource.GetName() != config.RuntimeConfigName {
		return
	}
	if err := ReconcileKiteGatewayExposureFromConfigMap(context.Background(), dynamicClient); err != nil {
		log.Printf("failed to reconcile Kite gateway exposure: %v", err)
	}
}

// reconcileGatewayExposureServiceEvent reconciles status when kite-gateway-external changes.
// dynamicClient applies the latest runtime config again, which keeps status accurate after LoadBalancer updates.
// obj is a Service informer event object or a deleted-final-state wrapper.
func reconcileGatewayExposureServiceEvent(dynamicClient dynamic.Interface, obj interface{}) {
	resource, ok := informerEventObject(obj)
	if !ok {
		log.Printf("Kite gateway exposure Service event is not an unstructured object")
		return
	}
	if resource.GetName() != kiteGatewayExternalServiceName {
		return
	}
	if err := ReconcileKiteGatewayExposureFromConfigMap(context.Background(), dynamicClient); err != nil {
		log.Printf("failed to reconcile Kite gateway exposure Service status: %v", err)
	}
}

func informerEventObject(obj interface{}) (*unstructured.Unstructured, bool) {
	resource, ok := obj.(*unstructured.Unstructured)
	if ok {
		return resource, true
	}
	deleted, ok := obj.(cache.DeletedFinalStateUnknown)
	if !ok {
		return nil, false
	}
	resource, ok = deleted.Obj.(*unstructured.Unstructured)
	return resource, ok
}

// gatewayBlockedStatus validates desired SSH gateway settings before Kubernetes resources are changed.
// desired is the operator-owned ConfigMap state.
// The boolean return is true when the external Service must stay absent.
func gatewayBlockedStatus(desired platform.SSHGatewayDesired) (platform.SSHGatewayStatus, bool) {
	if desired.ExternalEnabled && desired.ExternalPort == "" {
		return platform.SSHGatewayStatus{
			Phase:   platform.SSHGatewayPhaseBlocked,
			Reason:  platform.SSHGatewayReasonMissingExternalPort,
			Message: "External gateway is enabled but no user SSH port is configured.",
		}, true
	}
	if desired.HostFallbackEnabled && desired.HostSshdPort == "" {
		return platform.SSHGatewayStatus{
			Phase:   platform.SSHGatewayPhaseBlocked,
			Reason:  platform.SSHGatewayReasonMissingHostFallbackPort,
			Message: "Host fallback is enabled but no host sshd port is configured.",
		}, true
	}
	if desired.ExternalEnabled && desired.HostFallbackEnabled && desired.ExternalPort == desired.HostSshdPort {
		return platform.SSHGatewayStatus{
			Phase:   platform.SSHGatewayPhaseBlocked,
			Reason:  platform.SSHGatewayReasonPortConflict,
			Message: "External gateway port and host sshd port must be different.",
		}, true
	}
	return platform.SSHGatewayStatus{}, false
}

// normalizeInternalGatewayService keeps the internal kite-gateway Service as ClusterIP.
// ctx controls Kubernetes API requests.
// This repairs older installs where the internal Service may have been exposed directly.
func normalizeInternalGatewayService(ctx context.Context, dynamicClient dynamic.Interface) error {
	service, err := dynamicClient.Resource(serviceGVR).Namespace(config.KiteNamespace).Get(ctx, kiteGatewayInternalServiceName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read internal gateway Service: %w", err)
	}
	serviceType, _, _ := unstructured.NestedString(service.Object, "spec", "type")
	if serviceType == "" || serviceType == "ClusterIP" {
		return nil
	}
	patch := []map[string]any{
		{
			"op":    "replace",
			"path":  "/spec/type",
			"value": "ClusterIP",
		},
	}
	ports, _, _ := unstructured.NestedSlice(service.Object, "spec", "ports")
	for index, port := range ports {
		portMap, ok := port.(map[string]any)
		if !ok {
			continue
		}
		if _, exists := portMap["nodePort"]; exists {
			patch = append(patch, map[string]any{
				"op":   "remove",
				"path": fmt.Sprintf("/spec/ports/%d/nodePort", index),
			})
		}
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal internal gateway Service normalization patch: %w", err)
	}
	_, err = dynamicClient.Resource(serviceGVR).Namespace(config.KiteNamespace).Patch(ctx, kiteGatewayInternalServiceName, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to normalize internal gateway Service: %w", err)
	}
	return nil
}

// applyExternalGatewayService applies kite-gateway-external as a LoadBalancer Service.
// ctx controls the Kubernetes patch request.
// externalPort is the user-facing SSH port selected by a Level 3 admin.
func applyExternalGatewayService(ctx context.Context, dynamicClient dynamic.Interface, externalPort string) error {
	port, err := strconv.Atoi(externalPort)
	if err != nil {
		return fmt.Errorf("invalid external gateway port %q: %w", externalPort, err)
	}
	service := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name":      kiteGatewayExternalServiceName,
				"namespace": config.KiteNamespace,
				"labels": map[string]any{
					"app": "kite-gateway",
				},
			},
			"spec": map[string]any{
				"type": "LoadBalancer",
				"selector": map[string]any{
					"app": "kite-gateway",
				},
				"ports": []any{
					map[string]any{
						"name":       "ssh",
						"port":       int64(port),
						"targetPort": "ssh",
						"protocol":   "TCP",
					},
				},
			},
		},
	}
	data, err := json.Marshal(service.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal external gateway Service: %w", err)
	}
	_, err = dynamicClient.Resource(serviceGVR).Namespace(config.KiteNamespace).Patch(ctx, kiteGatewayExternalServiceName, types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: kiteGatewayApplyManager,
		Force:        ptr.To(true),
	})
	if err != nil {
		return fmt.Errorf("failed to apply external gateway Service: %w", err)
	}
	return nil
}

// externalGatewayServiceStatus reads the applied external Service and returns the admin-facing status.
// ctx controls the Kubernetes get request.
// externalPort and fallbackAddress are the desired values written into observed status fields.
func externalGatewayServiceStatus(ctx context.Context, dynamicClient dynamic.Interface, externalPort string, fallbackAddress string) (platform.SSHGatewayStatus, error) {
	service, err := dynamicClient.Resource(serviceGVR).Namespace(config.KiteNamespace).Get(ctx, kiteGatewayExternalServiceName, metav1.GetOptions{})
	if err != nil {
		return platform.SSHGatewayStatus{}, fmt.Errorf("failed to read external gateway Service status: %w", err)
	}
	return externalGatewayServiceStatusFromObject(service, externalPort, fallbackAddress), nil
}

func externalGatewayServiceStatusFromObject(service *unstructured.Unstructured, externalPort string, fallbackAddress string) platform.SSHGatewayStatus {
	status := platform.SSHGatewayStatus{
		ObservedExternalPort:        externalPort,
		ObservedHostFallbackAddress: fallbackAddress,
		ObservedServiceName:         kiteGatewayExternalServiceName,
	}
	ingress, _, _ := unstructured.NestedSlice(service.Object, "status", "loadBalancer", "ingress")
	if len(ingress) == 0 {
		status.Phase = platform.SSHGatewayPhaseReconciling
		status.Reason = platform.SSHGatewayReasonServicePending
		status.Message = "External VM SSH gateway Service was applied, but the LoadBalancer is not ready yet. If this remains pending, check whether the requested port is already used by the host or load balancer."
		return status
	}
	status.Phase = platform.SSHGatewayPhaseReady
	status.Reason = platform.SSHGatewayReasonServiceApplied
	status.Message = "External VM SSH gateway LoadBalancer is ready."
	return status
}

func deleteExternalGatewayService(ctx context.Context, dynamicClient dynamic.Interface) error {
	err := dynamicClient.Resource(serviceGVR).Namespace(config.KiteNamespace).Delete(ctx, kiteGatewayExternalServiceName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete external gateway Service: %w", err)
	}
	return nil
}

// patchGatewayFallback updates gateway Deployment environment variables for host fallback.
// ctx controls the Deployment update.
// NotFound is ignored because install ordering may create config before the Deployment.
func patchGatewayFallback(ctx context.Context, dynamicClient dynamic.Interface, enabled bool, address string) error {
	deployment, err := dynamicClient.Resource(deploymentGVR).Namespace(config.KiteNamespace).Get(ctx, kiteGatewayDeploymentName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read gateway Deployment: %w", err)
	}
	containers, _, _ := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
	for index, item := range containers {
		container, ok := item.(map[string]any)
		if !ok || container["name"] != kiteGatewayContainerName {
			continue
		}
		env, _, _ := unstructured.NestedSlice(container, "env")
		env = setGatewayEnv(env, "KITE_GATEWAY_HOST_FALLBACK_ENABLED", strconv.FormatBool(enabled))
		env = setGatewayEnv(env, "KITE_GATEWAY_HOST_SSHD_ADDRESS", address)
		container["env"] = env
		containers[index] = container
		if err := unstructured.SetNestedSlice(deployment.Object, containers, "spec", "template", "spec", "containers"); err != nil {
			return err
		}
		_, err = dynamicClient.Resource(deploymentGVR).Namespace(config.KiteNamespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update gateway Deployment fallback env: %w", err)
		}
		return nil
	}
	return fmt.Errorf("gateway Deployment container %q was not found", kiteGatewayContainerName)
}

// setGatewayEnv sets or appends one simple string env var entry.
// env is the existing container env slice from an unstructured Deployment.
// The returned slice is safe to put back into the container spec.
func setGatewayEnv(env []any, name string, value string) []any {
	entry := map[string]any{
		"name":  name,
		"value": value,
	}
	for index, item := range env {
		current, ok := item.(map[string]any)
		if ok && current["name"] == name {
			env[index] = entry
			return env
		}
	}
	return append(env, entry)
}

// writeGatewayFailedStatus records a failed reconcile attempt before returning the original error.
// ctx controls the status write.
// statusWriter writes kite-gateway-status.
func writeGatewayFailedStatus(ctx context.Context, statusWriter *platform.Service, err error) error {
	statusErr := statusWriter.WriteSSHGatewayStatus(ctx, withTransitionTime(platform.SSHGatewayStatus{
		Phase:   platform.SSHGatewayPhaseFailed,
		Reason:  platform.SSHGatewayReasonApplyFailed,
		Message: err.Error(),
	}))
	if statusErr != nil {
		return fmt.Errorf("%w; additionally failed to write gateway status: %v", err, statusErr)
	}
	return err
}

func withTransitionTime(status platform.SSHGatewayStatus) platform.SSHGatewayStatus {
	status.LastTransitionTime = time.Now().UTC().Format(time.RFC3339)
	return status
}
