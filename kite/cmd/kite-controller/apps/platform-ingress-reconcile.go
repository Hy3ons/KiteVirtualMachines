package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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
	platformhttpredirect "kite/internal/render/platform-http-redirect"
	platformhttpsredirect "kite/internal/render/platform-https-redirect"
	platformingress "kite/internal/render/platform-ingress"
)

const (
	kitePlatformIngressApplyManager = "kite-controller-platform-ingress"
)

var traefikMiddlewareGVR = schema.GroupVersionResource{
	Group:    "traefik.io",
	Version:  "v1alpha1",
	Resource: "middlewares",
}

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
	forceHTTPS, _, _ := unstructured.NestedString(configMap.Object, "data", config.ForceHTTPSConfigKey)
	shouldForceHTTPS := strings.EqualFold(forceHTTPS, "true")

	ingressObject, err := (&platformingress.PlatformIngressData{
		Namespace:  kiteGlobalConfigNamespace,
		Host:       host,
		ForceHTTPS: shouldForceHTTPS,
	}).Render()
	if err != nil {
		return fmt.Errorf("failed to render Kite platform Ingress: %w", err)
	}

	if shouldForceHTTPS {
		redirectObject, err := (&platformhttpsredirect.PlatformHTTPSRedirectData{
			Namespace: kiteGlobalConfigNamespace,
		}).Render()
		if err != nil {
			return fmt.Errorf("failed to render Kite platform HTTPS redirect middleware: %w", err)
		}
		if err := applyKitePlatformHTTPSRedirect(ctx, dynamicClient, redirectObject); err != nil {
			return err
		}

		redirectIngressObject, err := (&platformhttpredirect.PlatformHTTPRedirectData{
			Namespace: kiteGlobalConfigNamespace,
			Host:      host,
		}).Render()
		if err != nil {
			return fmt.Errorf("failed to render Kite platform HTTP redirect Ingress: %w", err)
		}
		if err := applyKitePlatformIngress(ctx, dynamicClient, redirectIngressObject); err != nil {
			return err
		}
		return applyKitePlatformIngress(ctx, dynamicClient, ingressObject)
	}

	if err := applyKitePlatformIngress(ctx, dynamicClient, ingressObject); err != nil {
		return err
	}
	return deleteKitePlatformHTTPSRedirect(ctx, dynamicClient)
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

// applyKitePlatformHTTPSRedirect applies the Traefik redirect middleware with server-side apply.
// ctx controls the Kubernetes API patch request.
// dynamicClient writes the Middleware object in the kite namespace.
// middlewareObject must be a traefik.io/v1alpha1 Middleware.
func applyKitePlatformHTTPSRedirect(ctx context.Context, dynamicClient dynamic.Interface, middlewareObject *unstructured.Unstructured) error {
	data, err := json.Marshal(middlewareObject.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal Kite platform HTTPS redirect middleware: %w", err)
	}

	_, err = dynamicClient.Resource(traefikMiddlewareGVR).Namespace(middlewareObject.GetNamespace()).Patch(ctx, middlewareObject.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: kitePlatformIngressApplyManager,
		Force:        ptr.To(true),
	})
	if err != nil {
		return fmt.Errorf("failed to apply Kite platform HTTPS redirect middleware %s/%s: %w", middlewareObject.GetNamespace(), middlewareObject.GetName(), err)
	}

	return nil
}

// ctx controls Kubernetes delete requests.
// dynamicClient deletes the HTTP redirect Ingress and then best-effort deletes Traefik Middleware.
// Missing resources are ignored, and Middleware cleanup errors are logged because the main HTTP Ingress must stay available for first-time setup.
func deleteKitePlatformHTTPSRedirect(ctx context.Context, dynamicClient dynamic.Interface) error {
	err := dynamicClient.Resource(ingressGVR).Namespace(kiteGlobalConfigNamespace).Delete(ctx, "kite-platform-http-redirect", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Kite platform HTTPS redirect Ingress: %w", err)
	}

	err = dynamicClient.Resource(traefikMiddlewareGVR).Namespace(kiteGlobalConfigNamespace).Delete(ctx, "kite-platform-https-redirect", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Printf("failed to delete Kite platform HTTPS redirect middleware: %v", err)
	}

	return nil
}
