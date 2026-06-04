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

// RunKiteVirtualMachineServiceReconciler watches VM-owned Service changes.
// clientManager provides the dynamic Kubernetes client used to watch core/v1 services.
// stopCh stops the informer during controller shutdown.
// This function is used so VM-owned Service drift can trigger KiteVirtualMachine reconciliation.
func RunKiteVirtualMachineServiceReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KiteVirtualMachine Service reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, time.Second*30)
	informer := factory.ForResource(serviceGVR).Informer()
	RegisterKiteVirtualMachineServiceReconciler(informer, clientManager.DynamicClient)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync Service informer cache")
		return
	}

	<-stopCh
}

// RegisterKiteVirtualMachineServiceReconciler attaches handlers to a Service informer.
// informer watches core/v1 Service resources across namespaces.
// dynamicClient reads the owning KiteVirtualMachine and refreshes VM status after Service changes.
// This function is called by controller startup code.
func RegisterKiteVirtualMachineServiceReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := ReconcileKiteVirtualMachineService(context.Background(), dynamicClient, obj, false); err != nil {
				log.Printf("failed to reconcile Kite VM Service add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := ReconcileKiteVirtualMachineService(context.Background(), dynamicClient, newObj, false); err != nil {
				log.Printf("failed to reconcile Kite VM Service update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := ReconcileKiteVirtualMachineService(context.Background(), dynamicClient, obj, true); err != nil {
				log.Printf("failed to reconcile Kite VM Service delete event: %v", err)
			}
		},
	})
}

// ReconcileKiteVirtualMachineService syncs VM-owned Service state into the owning KiteVirtualMachine.
// ctx controls Kubernetes API calls made during owner lookup and status update.
// dynamicClient reads KiteVirtualMachine resources after VM-owned Service changes.
// eventObj is a Service informer object or delete tombstone.
func ReconcileKiteVirtualMachineService(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}, deleted bool) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is required for Kite VM Service reconcile")
	}

	service, err := serviceFromEventObject(eventObj)
	if err != nil {
		return err
	}

	labels := service.GetLabels()
	if labels[kiteManagedByLabel] != kiteControllerLabel {
		return nil
	}

	kiteVMNamespace := labels[kiteVMNamespaceLabel]
	kiteVMName := labels[kiteVMNameLabel]
	if kiteVMNamespace == "" || kiteVMName == "" {
		return nil
	}

	kiteVMObject, err := dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(kiteVMNamespace).Get(ctx, kiteVMName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read owning KiteVirtualMachine %s/%s: %w", kiteVMNamespace, kiteVMName, err)
	}

	if deleted {
		return ReconcileKiteVirtualMachine(ctx, dynamicClient, kiteVMObject)
	}

	kiteVM, err := kiteVirtualMachineFromUnstructured(kiteVMObject)
	if err != nil {
		return err
	}

	domain, err := kiteVirtualMachineDomain(ctx, dynamicClient, kiteVM)
	if err != nil {
		return err
	}

	phase, _, _ := unstructured.NestedString(kiteVMObject.Object, "status", "phase")
	if phase == "" {
		phase = kiteVMPhaseProvisioning
	}
	powerState, _, _ := unstructured.NestedString(kiteVMObject.Object, "status", "currentPowerState")
	if powerState == "" {
		powerState = currentPowerStateFromVM(kiteVM)
	}

	return updateKiteVirtualMachineStatus(ctx, dynamicClient, kiteVM, phase, powerState, domain, metav1.ConditionFalse, kiteVMReasonReconciled, "VM Service is synced")
}

// serviceFromEventObject extracts a Service object from an informer event.
// eventObj may be a direct unstructured object or a DeletedFinalStateUnknown tombstone.
// The returned object is inspected for Kite ownership labels.
func serviceFromEventObject(eventObj interface{}) (*unstructured.Unstructured, error) {
	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		if tombstone, tombstoneOK := eventObj.(cache.DeletedFinalStateUnknown); tombstoneOK {
			resource, ok = tombstone.Obj.(*unstructured.Unstructured)
		}
	}
	if !ok {
		return nil, fmt.Errorf("Service event object is not unstructured")
	}

	return resource, nil
}
