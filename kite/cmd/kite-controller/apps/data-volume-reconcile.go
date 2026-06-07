package apps

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	kite "kite/api/v1"
	"kite/internal/kube"
)

const (
	kiteVMReasonDataVolumeReady    = "DataVolumeReady"
	kiteVMReasonDataVolumeProgress = "DataVolumeProgress"
	kiteVMReasonDataVolumeFailed   = "DataVolumeFailed"
	dataVolumePhaseSucceeded       = "Succeeded"
	dataVolumePhaseFailed          = "Failed"
)

// RunKiteVirtualMachineDataVolumeReconciler watches VM-owned DataVolume changes.
// clientManager provides the dynamic Kubernetes client used to watch CDI DataVolume resources.
// stopCh stops the informer during controller shutdown.
// This function is used to mirror disk import or clone progress into KiteVirtualMachine status.
func RunKiteVirtualMachineDataVolumeReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KiteVirtualMachine DataVolume reconciler requires a dynamic Kubernetes client")
		return
	}

	available, err := optionalResourceAvailable(clientManager, dataVolumeGVR)
	if err != nil {
		log.Printf("failed to discover DataVolume resource: %v", err)
		return
	}
	if !available {
		log.Printf("DataVolume resource %s is not installed; skipping DataVolume status watcher", dataVolumeGVR.String())
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, time.Second*30)
	informer := factory.ForResource(dataVolumeGVR).Informer()
	RegisterKiteVirtualMachineDataVolumeReconciler(informer, clientManager.DynamicClient)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync DataVolume informer cache")
		return
	}

	<-stopCh
}

// RegisterKiteVirtualMachineDataVolumeReconciler attaches handlers to a DataVolume informer.
// informer watches cdi.kubevirt.io/v1beta1 DataVolume resources across namespaces.
// dynamicClient reads the owning KiteVirtualMachine and updates its status.
// This function is called by controller startup code.
func RegisterKiteVirtualMachineDataVolumeReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := ReconcileKiteVirtualMachineDataVolume(context.Background(), dynamicClient, obj, false); err != nil {
				log.Printf("failed to reconcile DataVolume add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := ReconcileKiteVirtualMachineDataVolume(context.Background(), dynamicClient, newObj, false); err != nil {
				log.Printf("failed to reconcile DataVolume update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := ReconcileKiteVirtualMachineDataVolume(context.Background(), dynamicClient, obj, true); err != nil {
				log.Printf("failed to reconcile DataVolume delete event: %v", err)
			}
		},
	})
}

// ReconcileKiteVirtualMachineDataVolume mirrors DataVolume status into the owning KiteVirtualMachine.
// ctx controls Kubernetes API calls for owner lookup and status update.
// dynamicClient reads KiteVirtualMachine resources and updates their status subresource.
// eventObj is a DataVolume informer object or delete tombstone.
func ReconcileKiteVirtualMachineDataVolume(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}, deleted bool) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is required for DataVolume reconcile")
	}

	dataVolume, err := dataVolumeFromEventObject(eventObj)
	if err != nil {
		return err
	}

	labels := dataVolume.GetLabels()
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

	phase, progress, message, overallPhase, conditionStatus, reason := dataVolumeStatus(dataVolume)
	return updateKiteVirtualMachineDataVolumeStatus(ctx, dynamicClient, kiteVM, phase, progress, message, overallPhase, conditionStatus, reason)
}

// dataVolumeFromEventObject extracts a DataVolume object from an informer event.
// eventObj may be a direct unstructured object or a DeletedFinalStateUnknown tombstone.
// The returned object is inspected for Kite ownership labels and CDI status fields.
func dataVolumeFromEventObject(eventObj interface{}) (*unstructured.Unstructured, error) {
	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		if tombstone, tombstoneOK := eventObj.(cache.DeletedFinalStateUnknown); tombstoneOK {
			resource, ok = tombstone.Obj.(*unstructured.Unstructured)
		}
	}
	if !ok {
		return nil, fmt.Errorf("DataVolume event object is not unstructured")
	}

	return resource, nil
}

// dataVolumeStatus converts CDI DataVolume status into Kite status values.
// dataVolume is the unstructured cdi.kubevirt.io DataVolume object.
// The returned values are written to KiteVirtualMachine status.
func dataVolumeStatus(dataVolume *unstructured.Unstructured) (string, string, string, string, metav1.ConditionStatus, string) {
	phase, _, _ := unstructured.NestedString(dataVolume.Object, "status", "phase")
	phase = strings.TrimSpace(phase)
	if phase == "" {
		phase = "Pending"
	}

	progress, _, _ := unstructured.NestedString(dataVolume.Object, "status", "progress")
	message := dataVolumeStatusMessage(dataVolume)
	if message == "" {
		message = "DataVolume phase is " + phase
	}

	switch phase {
	case dataVolumePhaseSucceeded:
		return phase, progress, message, kiteVMPhaseProvisioning, metav1.ConditionFalse, kiteVMReasonDataVolumeReady
	case dataVolumePhaseFailed:
		return phase, progress, message, kiteVMPhaseFailed, metav1.ConditionFalse, kiteVMReasonDataVolumeFailed
	default:
		return phase, progress, message, kiteVMPhaseProvisioning, metav1.ConditionFalse, kiteVMReasonDataVolumeProgress
	}
}

// dataVolumeStatusMessage returns the most useful human-readable message from DataVolume conditions.
// dataVolume is the unstructured CDI DataVolume object.
// The returned string is empty when CDI has not reported condition details yet.
func dataVolumeStatusMessage(dataVolume *unstructured.Unstructured) string {
	conditions, _, _ := unstructured.NestedSlice(dataVolume.Object, "status", "conditions")
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if !ok {
			continue
		}
		message := machineStringValue(condition["message"])
		if message != "" {
			return message
		}
	}

	return ""
}

// updateKiteVirtualMachineDataVolumeStatus writes DataVolume fields into one KiteVirtualMachine status.
// ctx controls Kubernetes API get and status update calls.
// dynamicClient updates hy3ons.github.io/v1 kitevirtualmachines through the status subresource.
// vm identifies the CRD whose DataVolume status should be updated.
func updateKiteVirtualMachineDataVolumeStatus(ctx context.Context, dynamicClient dynamic.Interface, vm *kite.KiteVirtualMachine, dataVolumePhase string, dataVolumeProgress string, dataVolumeMessage string, overallPhase string, conditionStatus metav1.ConditionStatus, reason string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(vm.Namespace).Get(ctx, vm.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read KiteVirtualMachine %s/%s before DataVolume status update: %w", vm.Namespace, vm.Name, err)
		}

		observedGeneration := current.GetGeneration()
		if kiteVirtualMachineDataVolumeStatusMatches(current, observedGeneration, dataVolumePhase, dataVolumeProgress, dataVolumeMessage, overallPhase, conditionStatus, reason) {
			return nil
		}

		next := current.DeepCopy()
		if err := unstructured.SetNestedField(next.Object, observedGeneration, "status", "observedGeneration"); err != nil {
			return err
		}
		if err := unstructured.SetNestedField(next.Object, dataVolumePhase, "status", "dataVolumePhase"); err != nil {
			return err
		}
		if dataVolumeProgress != "" {
			if err := unstructured.SetNestedField(next.Object, dataVolumeProgress, "status", "dataVolumeProgress"); err != nil {
				return err
			}
		} else {
			unstructured.RemoveNestedField(next.Object, "status", "dataVolumeProgress")
		}
		if err := unstructured.SetNestedField(next.Object, dataVolumeMessage, "status", "dataVolumeMessage"); err != nil {
			return err
		}

		currentPhase, _, _ := unstructured.NestedString(next.Object, "status", "phase")
		if overallPhase == kiteVMPhaseFailed || currentPhase == "" || currentPhase == kiteVMPhaseProvisioning {
			if err := unstructured.SetNestedField(next.Object, overallPhase, "status", "phase"); err != nil {
				return err
			}
		}

		conditions, _, _ := unstructured.NestedSlice(next.Object, "status", "conditions")
		conditions = replaceKiteVirtualMachineCondition(conditions, kiteVirtualMachineCondition(observedGeneration, conditionStatus, reason, dataVolumeMessage))
		if err := unstructured.SetNestedSlice(next.Object, conditions, "status", "conditions"); err != nil {
			return err
		}

		_, err = dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(vm.Namespace).UpdateStatus(ctx, next, metav1.UpdateOptions{
			FieldManager: kiteVirtualMachineStatusManager,
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update KiteVirtualMachine %s/%s DataVolume status: %w", vm.Namespace, vm.Name, err)
	}

	return nil
}

// kiteVirtualMachineDataVolumeStatusMatches checks whether a DataVolume status update would be a no-op.
// current is the latest KiteVirtualMachine object from the API server.
// The remaining parameters are the status values the DataVolume watcher wants to write.
// A true return avoids noisy status updates.
func kiteVirtualMachineDataVolumeStatusMatches(current *unstructured.Unstructured, observedGeneration int64, dataVolumePhase string, dataVolumeProgress string, dataVolumeMessage string, overallPhase string, conditionStatus metav1.ConditionStatus, reason string) bool {
	currentObservedGeneration, _, _ := unstructured.NestedInt64(current.Object, "status", "observedGeneration")
	currentDataVolumePhase, _, _ := unstructured.NestedString(current.Object, "status", "dataVolumePhase")
	currentDataVolumeProgress, _, _ := unstructured.NestedString(current.Object, "status", "dataVolumeProgress")
	currentDataVolumeMessage, _, _ := unstructured.NestedString(current.Object, "status", "dataVolumeMessage")
	currentPhase, _, _ := unstructured.NestedString(current.Object, "status", "phase")
	currentCondition := findKiteVirtualMachineCondition(current.Object)

	phaseMatches := currentPhase == overallPhase
	if overallPhase != kiteVMPhaseFailed && currentPhase != "" && currentPhase != kiteVMPhaseProvisioning {
		phaseMatches = true
	}

	return currentObservedGeneration == observedGeneration &&
		currentDataVolumePhase == dataVolumePhase &&
		currentDataVolumeProgress == dataVolumeProgress &&
		currentDataVolumeMessage == dataVolumeMessage &&
		phaseMatches &&
		machineStringValue(currentCondition["status"]) == string(conditionStatus) &&
		machineStringValue(currentCondition["reason"]) == reason &&
		machineStringValue(currentCondition["message"]) == dataVolumeMessage
}
