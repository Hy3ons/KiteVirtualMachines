package apps

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"kite/internal/kube"
)

const (
	userNetworkPolicyDenyFromOtherNamespaces = "deny-from-other-namespaces"
	userNetworkPolicyTenantIsolationEgress   = "tenant-isolation-egress"
)

// RunKiteUserBaseResourceReconciler runs informers for KiteUser-owned namespace resources.
// clientManager provides the dynamic Kubernetes client used to watch NetworkPolicy resources.
// stopCh stops the informer factory when the controller process shuts down.
// This function is used by cmd/kite-controller/main.go so deleted policies are recreated from KiteUser spec.
func RunKiteUserBaseResourceReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KiteUser base resource reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, userReconcileResyncPeriod)
	networkPolicyInformer := factory.ForResource(networkPolicyGVR).Informer()

	RegisterKiteUserBaseResourceReconciler(networkPolicyInformer, clientManager.DynamicClient, networkPolicyGVR)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, networkPolicyInformer.HasSynced) {
		log.Printf("failed to sync KiteUser base resource informer cache")
		return
	}

	<-stopCh
}

// RegisterKiteUserBaseResourceReconciler attaches handlers to one user base resource informer.
// informer watches NetworkPolicy objects across all namespaces.
// dynamicClient lists KiteUsers and reapplies the matching user's desired namespace resources.
// resource identifies which watched resource type the handler is receiving.
func RegisterKiteUserBaseResourceReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface, resource schema.GroupVersionResource) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := ReconcileKiteUserBaseResource(context.Background(), dynamicClient, resource, obj); err != nil {
				log.Printf("failed to reconcile KiteUser base resource add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := ReconcileKiteUserBaseResource(context.Background(), dynamicClient, resource, newObj); err != nil {
				log.Printf("failed to reconcile KiteUser base resource update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := ReconcileKiteUserBaseResource(context.Background(), dynamicClient, resource, obj); err != nil {
				log.Printf("failed to reconcile KiteUser base resource delete event: %v", err)
			}
		},
	})
}

// ReconcileKiteUserBaseResource restores desired KiteUser namespace resources after drift.
// ctx controls Kubernetes API calls made while listing KiteUsers and applying rendered resources.
// dynamicClient is used for unstructured KiteUser and NetworkPolicy access.
// resource identifies the watched NetworkPolicy resource type.
// eventObj is the informer object or delete tombstone for a namespaced base resource.
func ReconcileKiteUserBaseResource(ctx context.Context, dynamicClient dynamic.Interface, resource schema.GroupVersionResource, eventObj interface{}) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	obj, err := baseResourceFromEventObject(eventObj)
	if err != nil {
		return err
	}
	if !kiteUserManagedBaseResource(resource, obj) {
		return nil
	}

	users, err := kiteUsersForNamespace(ctx, dynamicClient, obj.GetNamespace())
	if err != nil {
		return err
	}
	for i := range users {
		if err := ReconcileKiteUser(ctx, dynamicClient, &users[i]); err != nil {
			return fmt.Errorf("failed to restore KiteUser base resources for %s after %s %s/%s drift: %w", users[i].GetName(), obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
		}
	}

	if len(users) > 0 {
		log.Printf("restored KiteUser base resources in namespace %s after %s %s drift", obj.GetNamespace(), obj.GetKind(), obj.GetName())
	}
	return nil
}

// baseResourceFromEventObject extracts an unstructured resource from informer events.
// eventObj may be a live object or cache.DeletedFinalStateUnknown tombstone.
// The returned object provides namespace and name values for KiteUser lookup.
func baseResourceFromEventObject(eventObj interface{}) (*unstructured.Unstructured, error) {
	if tombstone, ok := eventObj.(cache.DeletedFinalStateUnknown); ok {
		eventObj = tombstone.Obj
	}

	obj, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("KiteUser base resource event object is not unstructured")
	}

	return obj, nil
}

// kiteUserManagedBaseResource checks whether one namespaced object is managed by KiteUser reconcile.
// resource is the GVR watched by the informer.
// obj provides the metadata.name to compare with the fixed resource names rendered for every user namespace.
// The returned value prevents unrelated user namespace resources from triggering reconcile loops.
func kiteUserManagedBaseResource(resource schema.GroupVersionResource, obj *unstructured.Unstructured) bool {
	switch resource {
	case networkPolicyGVR:
		return obj.GetName() == userNetworkPolicyDenyFromOtherNamespaces ||
			obj.GetName() == userNetworkPolicyTenantIsolationEgress
	default:
		return false
	}
}

// kiteUsersForNamespace lists KiteUsers that point at one namespace.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient lists cluster-scoped KiteUser custom resources.
// namespace is compared with spec.namespace.
func kiteUsersForNamespace(ctx context.Context, dynamicClient dynamic.Interface, namespace string) ([]unstructured.Unstructured, error) {
	users, err := dynamicClient.Resource(kiteUserGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list KiteUsers for namespace %s reconcile: %w", namespace, err)
	}

	matches := make([]unstructured.Unstructured, 0)
	for _, user := range users.Items {
		userNamespace, _, _ := unstructured.NestedString(user.Object, "spec", "namespace")
		if userNamespace == namespace {
			matches = append(matches, user)
		}
	}

	return matches, nil
}
