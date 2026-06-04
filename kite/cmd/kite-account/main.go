package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	kitev1 "kite/api/v1"
	"kite/internal/hostaccount"
	"kite/internal/kube"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const (
	resyncPeriod            = time.Second * 30
	kiteManagedByLabel      = "kite.anacnu.com/managed-by"
	kiteVMNameLabel         = "kite.anacnu.com/kite-vm-name"
	kiteVMNamespaceLabel    = "kite.anacnu.com/kite-vm-namespace"
	kiteControllerLabel     = "kite-controller"
	sshServiceNamePrefix    = "vps-access-"
	defaultHostRoot         = "/host"
	defaultSSHServicePort   = int64(22)
	kiteAccountServiceName  = "kite-account"
	kiteAccountStartupError = "kite-account requires a Kubernetes dynamic client"
)

var (
	kiteVirtualMachineGVR = schema.GroupVersionResource{
		Group:    "anacnu.com",
		Version:  "v1",
		Resource: "kitevirtualmachines",
	}
	serviceGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	}
)

func main() {
	clientManager, err := kube.GetClientManager()
	if err != nil {
		log.Fatalf("failed to connect to Kubernetes cluster: %v", err)
	}
	if clientManager.DynamicClient == nil {
		log.Fatalf(kiteAccountStartupError)
	}

	hostRoot := strings.TrimSpace(os.Getenv("KITE_ACCOUNT_HOST_ROOT"))
	if hostRoot == "" {
		hostRoot = defaultHostRoot
	}

	manager := hostaccount.NewManager(hostRoot)
	stopCh := make(chan struct{})
	runAccountInformers(clientManager.DynamicClient, manager, stopCh)
}

// runAccountInformers starts the KiteVM and Service informers used by kite-account.
// dynamicClient watches KiteVirtualMachine and Service resources.
// manager performs the actual host Linux account reconciliation.
// stopCh stops both informers during process shutdown.
func runAccountInformers(dynamicClient dynamic.Interface, manager *hostaccount.Manager, stopCh <-chan struct{}) {
	if dynamicClient == nil {
		log.Printf(kiteAccountStartupError)
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, resyncPeriod)
	vmInformer := factory.ForResource(kiteVirtualMachineGVR).Informer()
	serviceInformer := factory.ForResource(serviceGVR).Informer()

	registerKiteVMHandler(vmInformer, dynamicClient, manager)
	registerServiceHandler(serviceInformer, dynamicClient, manager)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, vmInformer.HasSynced, serviceInformer.HasSynced) {
		log.Printf("failed to sync %s informer caches", kiteAccountServiceName)
		return
	}

	<-stopCh
}

// registerKiteVMHandler attaches reconcile handlers to KiteVirtualMachine events.
// informer watches all KiteVirtualMachine CRDs.
// dynamicClient reads the VM-owned SSH Service.
// manager reconciles or deletes host Linux accounts based on VM desired state.
func registerKiteVMHandler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface, manager *hostaccount.Manager) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := reconcileKiteVMEvent(context.Background(), dynamicClient, manager, obj, false); err != nil {
				log.Printf("failed to reconcile Kite account add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := reconcileKiteVMEvent(context.Background(), dynamicClient, manager, newObj, false); err != nil {
				log.Printf("failed to reconcile Kite account update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := reconcileKiteVMEvent(context.Background(), dynamicClient, manager, obj, true); err != nil {
				log.Printf("failed to reconcile Kite account delete event: %v", err)
			}
		},
	})
}

// registerServiceHandler attaches handlers to VM-owned Service events.
// informer watches all core/v1 Services.
// dynamicClient reads the owning KiteVirtualMachine.
// manager refreshes host proxy shells when the Service ClusterIP changes.
func registerServiceHandler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface, manager *hostaccount.Manager) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := reconcileServiceEvent(context.Background(), dynamicClient, manager, obj); err != nil {
				log.Printf("failed to reconcile Kite account Service add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := reconcileServiceEvent(context.Background(), dynamicClient, manager, newObj); err != nil {
				log.Printf("failed to reconcile Kite account Service update event: %v", err)
			}
		},
	})
}

// reconcileKiteVMEvent reconciles or deletes one host account from a KiteVirtualMachine event.
// ctx controls Kubernetes and host command calls.
// dynamicClient reads the matching SSH Service.
// manager applies host account changes.
// eventObj is a KiteVirtualMachine informer object or tombstone.
// deleted is true for Kubernetes delete events and forces account cleanup.
func reconcileKiteVMEvent(ctx context.Context, dynamicClient dynamic.Interface, manager *hostaccount.Manager, eventObj interface{}, deleted bool) error {
	vm, err := kiteVirtualMachineFromEvent(eventObj)
	if err != nil {
		return err
	}
	if vm.Namespace == "" || vm.Name == "" {
		return nil
	}
	if strings.TrimSpace(vm.Spec.SSHID) == "" {
		return nil
	}

	if deleted || vm.Spec.Delete {
		return manager.Delete(ctx, vm.Spec.SSHID, vm.Namespace, vm.Name)
	}

	clusterIP, port, err := sshServiceTarget(ctx, dynamicClient, vm.Namespace, vm.Name)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return manager.Ensure(ctx, hostaccount.DesiredAccount{
		Username:    vm.Spec.SSHID,
		Password:    vm.Spec.SSHPassword,
		VMNamespace: vm.Namespace,
		VMName:      vm.Name,
		ClusterIP:   clusterIP,
		Port:        port,
	})
}

// reconcileServiceEvent refreshes the account for the VM that owns one Service.
// ctx controls Kubernetes and host command calls.
// dynamicClient reads the owning KiteVirtualMachine.
// manager applies host account changes after Service ClusterIP changes.
// eventObj is a Service informer object.
func reconcileServiceEvent(ctx context.Context, dynamicClient dynamic.Interface, manager *hostaccount.Manager, eventObj interface{}) error {
	service, err := serviceFromEvent(eventObj)
	if err != nil {
		return err
	}

	labels := service.GetLabels()
	if labels[kiteManagedByLabel] != kiteControllerLabel {
		return nil
	}
	if !strings.HasPrefix(service.GetName(), sshServiceNamePrefix) {
		return nil
	}

	vmNamespace := labels[kiteVMNamespaceLabel]
	vmName := labels[kiteVMNameLabel]
	if vmNamespace == "" || vmName == "" {
		return nil
	}

	vmObject, err := dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(vmNamespace).Get(ctx, vmName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read KiteVirtualMachine %s/%s for account reconcile: %w", vmNamespace, vmName, err)
	}

	return reconcileKiteVMEvent(ctx, dynamicClient, manager, vmObject, false)
}

// sshServiceTarget reads the ClusterIP and SSH port from vps-access-<vmName>.
// ctx controls the Service get request.
// dynamicClient reads core/v1 Services.
// namespace and vmName identify the KiteVirtualMachine and fixed SSH Service name.
func sshServiceTarget(ctx context.Context, dynamicClient dynamic.Interface, namespace string, vmName string) (string, int64, error) {
	service, err := dynamicClient.Resource(serviceGVR).Namespace(namespace).Get(ctx, sshServiceNamePrefix+vmName, metav1.GetOptions{})
	if err != nil {
		return "", 0, err
	}

	clusterIP, _, _ := unstructured.NestedString(service.Object, "spec", "clusterIP")
	if strings.TrimSpace(clusterIP) == "" || clusterIP == "None" {
		return "", 0, fmt.Errorf("SSH Service %s/%s has no ClusterIP", namespace, service.GetName())
	}

	ports, _, _ := unstructured.NestedSlice(service.Object, "spec", "ports")
	for _, item := range ports {
		port, ok := item.(map[string]any)
		if !ok {
			continue
		}
		value, ok := numericInt64(port["port"])
		if ok && value > 0 {
			return clusterIP, value, nil
		}
	}

	return clusterIP, defaultSSHServicePort, nil
}

// kiteVirtualMachineFromEvent extracts a typed KiteVirtualMachine from an informer event.
// eventObj may be a direct unstructured object or a delete tombstone.
// The returned VM provides spec.sshId, spec.sshPassword, namespace, and name for account reconcile.
func kiteVirtualMachineFromEvent(eventObj interface{}) (*kitev1.KiteVirtualMachine, error) {
	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		if tombstone, tombstoneOK := eventObj.(cache.DeletedFinalStateUnknown); tombstoneOK {
			resource, ok = tombstone.Obj.(*unstructured.Unstructured)
		}
	}
	if !ok {
		return nil, fmt.Errorf("KiteVirtualMachine event object is not unstructured")
	}

	var vm kitev1.KiteVirtualMachine
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
		return nil, err
	}
	return &vm, nil
}

// serviceFromEvent extracts an unstructured Service from an informer event.
// eventObj may be a direct object or a delete tombstone.
// The returned object is inspected for Kite VM ownership labels.
func serviceFromEvent(eventObj interface{}) (*unstructured.Unstructured, error) {
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

// numericInt64 converts Kubernetes unstructured numeric values to int64.
// value is read from Service spec.ports.
// The returned boolean is false when value is not numeric.
func numericInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case float64:
		return int64(typed), true
	default:
		return 0, false
	}
}
