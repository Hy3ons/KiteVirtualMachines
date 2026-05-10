package apps

import (
	"context"
	"fmt"
	"log"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"kite/internal/kube"
)

const namespaceReconcileResyncPeriod = time.Minute * 3

// RunKiteNamespaceReconciler creates and runs the Namespace informer loop.
// clientManager provides the dynamic Kubernetes client used to watch namespaces and list KiteUsers.
// stopCh stops the informer when the controller process is shutting down.
// This function is expected to run in a goroutine from cmd/kite-controller/main.go.
func RunKiteNamespaceReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("Kite namespace reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, namespaceReconcileResyncPeriod)
	informer := factory.ForResource(namespaceGVR).Informer()
	RegisterKiteNamespaceReconciler(informer, clientManager)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync Kite namespace informer cache")
		return
	}

	<-stopCh
}

// RegisterKiteNamespaceReconciler attaches Namespace event handlers to an informer.
// informer watches cluster-scoped Kubernetes Namespace resources from controller startup code.
// clientManager provides the dynamic Kubernetes client used to list KiteUsers and delete orphan namespaces.
// This function is used by cmd/kite-controller/main.go when wiring controller informers.
func RegisterKiteNamespaceReconciler(informer cache.SharedIndexInformer, clientManager *kube.ClientManager) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := ReconcileKiteNamespace(context.Background(), clientManager.DynamicClient, obj); err != nil {
				log.Printf("failed to reconcile namespace add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := ReconcileKiteNamespace(context.Background(), clientManager.DynamicClient, newObj); err != nil {
				log.Printf("failed to reconcile namespace update event: %v", err)
			}
		},
	})
}

// ReconcileKiteNamespace deletes Kite-managed namespaces that no KiteUser references.
// ctx controls the Kubernetes API calls made during reconciliation.
// dynamicClient lists KiteUser custom resources and deletes orphan Namespace resources.
// eventObj is the informer event object for a Namespace resource.
// This function is used by Namespace add and update event handlers.
func ReconcileKiteNamespace(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	namespace, err := namespaceFromEventObject(eventObj)
	if err != nil {
		return err
	}

	if !kiteManagedNamespace(namespace) {
		return nil
	}

	referenced, err := namespaceReferencedByKiteUser(ctx, dynamicClient, namespace.GetName())
	if err != nil {
		return err
	}
	if referenced {
		return nil
	}

	if err := dynamicClient.Resource(namespaceGVR).Delete(ctx, namespace.GetName(), metav1.DeleteOptions{}); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete orphan Kite namespace %s: %w", namespace.GetName(), err)
	}

	log.Printf("deleted orphan Kite namespace %s", namespace.GetName())
	return nil
}

// namespaceReferencedByKiteUser checks whether any KiteUser points at one namespace.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient lists cluster-scoped KiteUser custom resources.
// namespace is the Namespace metadata.name to compare with spec.namespace.
// The returned value is true when at least one KiteUser still references the namespace.
func namespaceReferencedByKiteUser(ctx context.Context, dynamicClient dynamic.Interface, namespace string) (bool, error) {
	users, err := dynamicClient.Resource(kiteUserGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list KiteUsers for namespace %s cleanup: %w", namespace, err)
	}

	for _, user := range users.Items {
		userNamespace, _, _ := unstructured.NestedString(user.Object, "spec", "namespace")
		if userNamespace == namespace {
			return true, nil
		}
	}

	return false, nil
}

// namespaceFromEventObject converts an informer event object into a Namespace resource.
// eventObj can be an unstructured Namespace or a DeletedFinalStateUnknown tombstone.
// The returned object contains metadata labels used to decide whether Kite manages it.
// This helper is used by Namespace add and update event handlers.
func namespaceFromEventObject(eventObj interface{}) (*unstructured.Unstructured, error) {
	if tombstone, ok := eventObj.(cache.DeletedFinalStateUnknown); ok {
		eventObj = tombstone.Obj
	}

	namespace, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("namespace event object is not unstructured")
	}

	return namespace, nil
}
