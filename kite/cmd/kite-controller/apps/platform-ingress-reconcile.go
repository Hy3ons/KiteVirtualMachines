package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"

	"kite/internal/kube"
	platformingress "kite/internal/render/platform-ingress"
)

const (
	kitePlatformIngressApplyManager = "kite-controller-platform-ingress"
)

// RunKitePlatformIngressReconciler watches kite-runtime-config and reconciles the platform Ingress.
// clientManager provides the dynamic Kubernetes client.
// stopCh stops the informer when the controller process is shutting down.
// This function is started by cmd/kite-controller/main.go.
func RunKitePlatformIngressReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("Kite platform Ingress reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		clientManager.DynamicClient,
		time.Second*30,
		kiteGlobalConfigNamespace,
		nil,
	)
	informer := factory.ForResource(configMapGVR).Informer()
	RegisterKitePlatformIngressReconciler(informer, clientManager.DynamicClient)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync Kite platform Ingress informer cache")
		return
	}

	if err := ReconcileKitePlatformIngressFromConfigMap(context.Background(), clientManager.DynamicClient); err != nil {
		log.Printf("failed to reconcile initial Kite platform Ingress: %v", err)
	}

	<-stopCh
}

// RegisterKitePlatformIngressReconciler attaches ConfigMap event handlers.
// informer watches ConfigMaps in the kite namespace.
// dynamicClient applies the platform Ingress after runtime config changes.
// This function is used by RunKitePlatformIngressReconciler.
func RegisterKitePlatformIngressReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			reconcilePlatformIngressConfigEvent(dynamicClient, obj)
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			reconcilePlatformIngressConfigEvent(dynamicClient, newObj)
		},
	})
}

// ReconcileKitePlatformIngressFromConfigMap reads runtime config and applies kite-platform Ingress.
// ctx controls Kubernetes API calls.
// dynamicClient reads kite/kite-runtime-config and writes networking.k8s.io Ingress.
// A nil error means the Ingress was applied or runtime config is not available yet.
func ReconcileKitePlatformIngressFromConfigMap(ctx context.Context, dynamicClient dynamic.Interface) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is required for Kite platform Ingress reconcile")
	}

	configMap, err := dynamicClient.Resource(configMapGVR).Namespace(kiteGlobalConfigNamespace).Get(ctx, kiteGlobalConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	host, _, _ := unstructured.NestedString(configMap.Object, "data", kiteGlobalBaseDomainKey)
	host = strings.Trim(strings.TrimSpace(host), ".")

	ingressObject, err := (&platformingress.PlatformIngressData{
		Namespace: kiteGlobalConfigNamespace,
		Host:      host,
	}).Render()
	if err != nil {
		return fmt.Errorf("failed to render Kite platform Ingress: %w", err)
	}

	return applyKitePlatformIngress(ctx, dynamicClient, ingressObject)
}

// reconcilePlatformIngressConfigEvent reconciles platform ingress for runtime config changes.
// dynamicClient applies the rendered Ingress object.
// obj is a ConfigMap informer event object.
// Non-runtime ConfigMaps are ignored.
func reconcilePlatformIngressConfigEvent(dynamicClient dynamic.Interface, obj interface{}) {
	resource, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Printf("Kite platform Ingress event is not an unstructured ConfigMap")
		return
	}
	if resource.GetName() != kiteGlobalConfigName {
		return
	}

	if err := ReconcileKitePlatformIngressFromConfigMap(context.Background(), dynamicClient); err != nil {
		log.Printf("failed to reconcile Kite platform Ingress: %v", err)
	}
}

// applyKitePlatformIngress applies the rendered platform Ingress with server-side apply.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient writes the Ingress object in the kite namespace.
// ingressObject must be a networking.k8s.io/v1 Ingress.
func applyKitePlatformIngress(ctx context.Context, dynamicClient dynamic.Interface, ingressObject *unstructured.Unstructured) error {
	data, err := json.Marshal(ingressObject.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal Kite platform Ingress: %w", err)
	}

	_, err = dynamicClient.Resource(ingressGVR).Namespace(ingressObject.GetNamespace()).Patch(ctx, ingressObject.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: kitePlatformIngressApplyManager,
		Force:        ptr.To(true),
	})
	if err != nil {
		return fmt.Errorf("failed to apply Kite platform Ingress %s/%s: %w", ingressObject.GetNamespace(), ingressObject.GetName(), err)
	}

	return nil
}
