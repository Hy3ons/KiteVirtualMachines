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
)

// RunKiteGatewayExposureReconciler watches runtime config and reconciles SSH gateway exposure.
// clientManager provides the dynamic Kubernetes client.
// stopCh stops the informer during controller shutdown.
// This reconciler owns only kite-gateway-external.
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
// dynamicClient reads desired config, writes status, and applies the external Service.
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
		return statusWriter.WriteSSHGatewayStatus(ctx, withTransitionTime(status))
	}

	if !desired.ExternalEnabled {
		if err := deleteExternalGatewayService(ctx, dynamicClient); err != nil {
			return writeGatewayFailedStatus(ctx, statusWriter, err)
		}
		return statusWriter.WriteSSHGatewayStatus(ctx, withTransitionTime(platform.SSHGatewayStatus{
			Phase:   platform.SSHGatewayPhaseDisabled,
			Reason:  platform.SSHGatewayReasonExternalDisabled,
			Message: "외부 VM SSH gateway가 비활성화되어 있습니다. VM SSH 접속을 열려면 Admin Settings에서 Service 포트와 사용자 안내 포트를 설정하세요.",
		}))
	}

	if err := applyExternalGatewayService(ctx, dynamicClient, desired.ExternalPort); err != nil {
		return writeGatewayFailedStatus(ctx, statusWriter, err)
	}

	appliedStatus, err := externalGatewayServiceStatus(ctx, dynamicClient, desired.ExternalPort)
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
			Message: "외부 VM SSH gateway가 켜져 있지만 Gateway Service 포트가 비어 있습니다. Admin Settings에서 Kubernetes LoadBalancer Service가 열 포트를 입력하세요.",
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
// externalPort is the Service port selected by a Level 3 admin.
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
// externalPort is the desired Service port written into observed status.
func externalGatewayServiceStatus(ctx context.Context, dynamicClient dynamic.Interface, externalPort string) (platform.SSHGatewayStatus, error) {
	service, err := dynamicClient.Resource(serviceGVR).Namespace(config.KiteNamespace).Get(ctx, kiteGatewayExternalServiceName, metav1.GetOptions{})
	if err != nil {
		return platform.SSHGatewayStatus{}, fmt.Errorf("failed to read external gateway Service status: %w", err)
	}
	return externalGatewayServiceStatusFromObject(service, externalPort), nil
}

func externalGatewayServiceStatusFromObject(service *unstructured.Unstructured, externalPort string) platform.SSHGatewayStatus {
	status := platform.SSHGatewayStatus{
		ObservedExternalPort: externalPort,
		ObservedServiceName:  kiteGatewayExternalServiceName,
	}
	ingress, _, _ := unstructured.NestedSlice(service.Object, "status", "loadBalancer", "ingress")
	if len(ingress) == 0 {
		status.Phase = platform.SSHGatewayPhaseReconciling
		status.Reason = platform.SSHGatewayReasonServicePending
		status.Message = "Gateway Service는 적용됐지만 LoadBalancer 주소가 아직 준비되지 않았습니다. 오래 지속되면 Service 포트 사용 가능 여부와 클러스터 LoadBalancer 상태를 확인하세요."
		return status
	}
	status.Phase = platform.SSHGatewayPhaseReady
	status.Reason = platform.SSHGatewayReasonServiceApplied
	status.Message = "외부 VM SSH gateway가 준비되었습니다. 사용자는 Dashboard에 표시된 SSH 명령으로 VM에 접속할 수 있습니다."
	return status
}

func deleteExternalGatewayService(ctx context.Context, dynamicClient dynamic.Interface) error {
	err := dynamicClient.Resource(serviceGVR).Namespace(config.KiteNamespace).Delete(ctx, kiteGatewayExternalServiceName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete external gateway Service: %w", err)
	}
	return nil
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
