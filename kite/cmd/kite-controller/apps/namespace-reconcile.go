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
		DeleteFunc: func(obj interface{}) {
			if err := RestoreKiteNamespace(context.Background(), clientManager.DynamicClient, obj); err != nil {
				log.Printf("failed to restore Kite namespace after delete event: %v", err)
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

// RestoreKiteNamespace recreates a deleted Kite-managed namespace when a KiteUser still references it.
// ctx controls Kubernetes API calls made while listing KiteUsers and applying namespace resources.
// dynamicClient lists KiteUsers and applies the matching user's rendered base resources.
// eventObj is the deleted Namespace informer event object.
// This function is used by the Namespace delete handler to keep KiteUser desired state authoritative.
func RestoreKiteNamespace(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}) error {
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

	users, err := kiteUsersForNamespace(ctx, dynamicClient, namespace.GetName())
	if err != nil {
		return err
	}
	for i := range users {
		if err := ReconcileKiteUser(ctx, dynamicClient, &users[i]); err != nil {
			return fmt.Errorf("failed to restore Kite namespace %s for KiteUser %s: %w", namespace.GetName(), users[i].GetName(), err)
		}
	}

	if len(users) > 0 {
		log.Printf("restored deleted Kite namespace %s from KiteUser desired state", namespace.GetName())
	}
	return nil
}

// namespaceReferencedByKiteUser checks whether any KiteUser points at one namespace.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient lists cluster-scoped KiteUser custom resources.
// namespace is the Namespace metadata.name to compare with spec.namespace.
// The returned value is true when at least one KiteUser still references the namespace.
func namespaceReferencedByKiteUser(ctx context.Context, dynamicClient dynamic.Interface, namespace string) (bool, error) {
	users, err := kiteUsersForNamespace(ctx, dynamicClient, namespace)
	if err != nil {
		return false, err
	}

	return len(users) > 0, nil
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
