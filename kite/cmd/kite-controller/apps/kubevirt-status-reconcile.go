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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	kite "kite/api/v1"
	"kite/internal/kube"
)

const (
	kiteManagedByLabel     = "kite.anacnu.com/managed-by"
	kiteVMNameLabel        = "kite.anacnu.com/kite-vm-name"
	kiteVMNamespaceLabel   = "kite.anacnu.com/kite-vm-namespace"
	kiteControllerLabel    = "kite-controller"
	kubeVirtStatusUnknown  = "Unknown"
	kubeVirtStatusRunning  = "Running"
	kubeVirtStatusStarting = "Starting"
	kubeVirtStatusStopped  = "Stopped"
)

// RunKubeVirtVirtualMachineStatusReconciler watches KubeVirt VirtualMachine status changes.
// clientManager provides the dynamic Kubernetes client used to watch kubevirt.io/v1 virtualmachines.
// stopCh stops the informer during controller shutdown.
// This function is expected to run beside the KiteVirtualMachine reconciler in controller startup.
func RunKubeVirtVirtualMachineStatusReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KubeVirt VirtualMachine status reconciler requires a dynamic Kubernetes client")
		return
	}

	available, err := optionalResourceAvailable(clientManager, kubeVirtVirtualMachineGVR)
	if err != nil {
		log.Printf("failed to discover KubeVirt VirtualMachine resource: %v", err)
		return
	}
	if !available {
		log.Printf("KubeVirt VirtualMachine resource %s is not installed; skipping KubeVirt status watcher", kubeVirtVirtualMachineGVR.String())
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, time.Second*30)
	informer := factory.ForResource(kubeVirtVirtualMachineGVR).Informer()
	RegisterKubeVirtVirtualMachineStatusReconciler(informer, clientManager.DynamicClient)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync KubeVirt VirtualMachine informer cache")
		return
	}

	<-stopCh
}

// RegisterKubeVirtVirtualMachineStatusReconciler attaches status handlers to a KubeVirt VM informer.
// informer watches kubevirt.io/v1 VirtualMachine resources.
// dynamicClient reads the owning KiteVirtualMachine and updates its status.
// This function is used by RunKubeVirtVirtualMachineStatusReconciler.
func RegisterKubeVirtVirtualMachineStatusReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := ReconcileKubeVirtVirtualMachineStatus(context.Background(), dynamicClient, obj, false); err != nil {
				log.Printf("failed to reconcile KubeVirt VirtualMachine add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := ReconcileKubeVirtVirtualMachineStatus(context.Background(), dynamicClient, newObj, false); err != nil {
				log.Printf("failed to reconcile KubeVirt VirtualMachine update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := ReconcileKubeVirtVirtualMachineStatus(context.Background(), dynamicClient, obj, true); err != nil {
				log.Printf("failed to reconcile KubeVirt VirtualMachine delete event: %v", err)
			}
		},
	})
}

// ReconcileKubeVirtVirtualMachineStatus mirrors KubeVirt VM state into the owning KiteVirtualMachine CRD.
// ctx controls Kubernetes API calls made during status reads and writes.
// dynamicClient reads KiteVirtualMachine resources and updates their status.
// eventObj is a KubeVirt VirtualMachine informer object or delete tombstone.
func ReconcileKubeVirtVirtualMachineStatus(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}, deleted bool) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is required for KubeVirt status reconcile")
	}

	kubeVirtVM, err := kubeVirtVirtualMachineFromEventObject(eventObj)
	if err != nil {
		return err
	}

	labels := kubeVirtVM.GetLabels()
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

	kiteVM, err := kiteVirtualMachineFromUnstructured(kiteVMObject)
	if err != nil {
		return err
	}

	if deleted {
		return ReconcileKiteVirtualMachine(ctx, dynamicClient, kiteVMObject)
	}

	domain, err := kiteVirtualMachineDomain(ctx, dynamicClient, kiteVM)
	if err != nil {
		return err
	}

	phase, powerState, conditionStatus, message := kiteStatusFromKubeVirtVirtualMachine(kubeVirtVM)
	nodeName := kiteNodeNameFromKubeVirtVirtualMachine(kubeVirtVM)
	if err := updateKiteVirtualMachineStatus(ctx, dynamicClient, kiteVM, phase, powerState, domain, nodeName, conditionStatus, kiteVMReasonReconciled, message); err != nil {
		return err
	}

	if kiteVM.Spec.Delete || vmShouldRun(kiteVM) != kubeVirtVirtualMachineRunning(kubeVirtVM) {
		return ReconcileKiteVirtualMachine(ctx, dynamicClient, kiteVMObject)
	}

	return nil
}

// kiteNodeNameFromKubeVirtVirtualMachine reads the node currently reported by KubeVirt.
// kubeVirtVM is the kubevirt.io/v1 VirtualMachine object from the status informer.
// The returned value may be empty when KubeVirt has not surfaced node placement yet.
func kiteNodeNameFromKubeVirtVirtualMachine(kubeVirtVM *unstructured.Unstructured) string {
	nodeName, _, _ := unstructured.NestedString(kubeVirtVM.Object, "status", "nodeName")
	return strings.TrimSpace(nodeName)
}

// kubeVirtVirtualMachineFromEventObject extracts a KubeVirt VM unstructured object from an informer event.
// eventObj may be a direct unstructured object or a DeletedFinalStateUnknown tombstone.
// The returned object is used to find owner labels and KubeVirt status fields.
func kubeVirtVirtualMachineFromEventObject(eventObj interface{}) (*unstructured.Unstructured, error) {
	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		if tombstone, tombstoneOK := eventObj.(cache.DeletedFinalStateUnknown); tombstoneOK {
			resource, ok = tombstone.Obj.(*unstructured.Unstructured)
		}
	}
	if !ok {
		return nil, fmt.Errorf("KubeVirt VirtualMachine event object is not unstructured")
	}

	return resource, nil
}

// kiteVirtualMachineFromUnstructured converts a KiteVirtualMachine object into the local API struct.
// resource is the current unstructured CRD object from the Kubernetes API server.
// The returned struct is used by status and drift reconciliation logic.
func kiteVirtualMachineFromUnstructured(resource *unstructured.Unstructured) (*kite.KiteVirtualMachine, error) {
	var vm kite.KiteVirtualMachine
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
		return nil, err
	}

	return &vm, nil
}

// kiteStatusFromKubeVirtVirtualMachine converts KubeVirt VM status into Kite CRD status values.
// kubeVirtVM is the real kubevirt.io/v1 VirtualMachine object.
// The returned phase, powerState, condition status, and message are written to KiteVirtualMachine status.
func kiteStatusFromKubeVirtVirtualMachine(kubeVirtVM *unstructured.Unstructured) (string, string, metav1.ConditionStatus, string) {
	running := kubeVirtVirtualMachineRunning(kubeVirtVM)
	powerState := "Off"
	if running {
		powerState = "On"
	}

	printableStatus, _, _ := unstructured.NestedString(kubeVirtVM.Object, "status", "printableStatus")
	printableStatus = strings.TrimSpace(printableStatus)
	if printableStatus == "" {
		if statusPhase, _, _ := unstructured.NestedString(kubeVirtVM.Object, "status", "phase"); statusPhase != "" {
			printableStatus = statusPhase
		}
	}
	if printableStatus == "" {
		printableStatus = kubeVirtStatusUnknown
	}

	switch printableStatus {
	case kubeVirtStatusRunning:
		return kiteVMPhaseRunning, powerState, metav1.ConditionTrue, "KubeVirt VirtualMachine is running"
	case kubeVirtStatusStopped:
		return kiteVMPhaseStopped, powerState, metav1.ConditionTrue, "KubeVirt VirtualMachine is stopped"
	case kubeVirtStatusStarting:
		return kiteVMPhaseProvisioning, powerState, metav1.ConditionFalse, "KubeVirt VirtualMachine is starting"
	default:
		if running {
			return kiteVMPhaseProvisioning, powerState, metav1.ConditionFalse, "KubeVirt VirtualMachine status is " + printableStatus
		}

		return kiteVMPhaseReady, powerState, metav1.ConditionTrue, "KubeVirt VirtualMachine status is " + printableStatus
	}
}

// kubeVirtVirtualMachineRunning reads the KubeVirt VM running intent.
// kubeVirtVM is the unstructured kubevirt.io/v1 VirtualMachine object.
// A false return means the VM is intended to be powered off or the field is absent.
func kubeVirtVirtualMachineRunning(kubeVirtVM *unstructured.Unstructured) bool {
	runStrategy, _, _ := unstructured.NestedString(kubeVirtVM.Object, "spec", "runStrategy")
	switch runStrategy {
	case "Always", "RerunOnFailure", "Manual":
		return true
	case "Halted":
		return false
	}

	running, _, _ := unstructured.NestedBool(kubeVirtVM.Object, "spec", "running")
	return running
}
