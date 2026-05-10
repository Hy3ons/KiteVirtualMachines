package apps

import (
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	kite "kite/api/v1"
	"kite/internal/kube"
)

var kiteVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}

// RunKiteVirtualMachineReconciler creates and runs the KiteVirtualMachine informer loop.
// clientManager provides the dynamic Kubernetes client used to watch KiteVirtualMachine resources.
// stopCh stops the informer when the controller process is shutting down.
// This function is expected to run in a goroutine from cmd/kite-controller/main.go.
func RunKiteVirtualMachineReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KiteVirtualMachine reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, time.Second*30)
	informer := factory.ForResource(kiteVirtualMachineGVR).Informer()
	RegisterKiteVirtualMachineReconciler(informer)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync KiteVirtualMachine informer cache")
		return
	}

	<-stopCh
}

// RegisterKiteVirtualMachineReconciler attaches KiteVirtualMachine event handlers to an informer.
// informer watches namespaced KiteVirtualMachine custom resources from this controller app.
// This function currently logs VM events until the real VM reconcile flow is implemented.
func RegisterKiteVirtualMachineReconciler(informer cache.SharedIndexInformer) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			resource, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Printf("KiteVirtualMachine add event object is not unstructured")
				return
			}

			var vm kite.KiteVirtualMachine
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
				log.Printf("failed to convert KiteVirtualMachine add event: %v", err)
				return
			}

			log.Printf("KiteVirtualMachine added: %s/%s", resource.GetNamespace(), resource.GetName())
		},

		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldResource, ok := oldObj.(metav1.Object)
			if !ok {
				log.Printf("old KiteVirtualMachine update event object is not metav1.Object")
				return
			}

			newResource, ok := newObj.(metav1.Object)
			if !ok {
				log.Printf("new KiteVirtualMachine update event object is not metav1.Object")
				return
			}

			if oldResource.GetGeneration() == newResource.GetGeneration() {
				return
			}

			log.Printf("KiteVirtualMachine updated: %s/%s", newResource.GetNamespace(), newResource.GetName())
		},

		DeleteFunc: func(obj interface{}) {
			resource, ok := obj.(*unstructured.Unstructured)
			if !ok {
				if tombstone, tombstoneOK := obj.(cache.DeletedFinalStateUnknown); tombstoneOK {
					resource, ok = tombstone.Obj.(*unstructured.Unstructured)
				}
			}
			if !ok {
				log.Printf("KiteVirtualMachine delete event object is not unstructured")
				return
			}

			var vm kite.KiteVirtualMachine
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
				log.Printf("failed to convert KiteVirtualMachine delete event: %v", err)
				return
			}

			log.Printf("KiteVirtualMachine deleted: %s/%s", resource.GetNamespace(), resource.GetName())
		},
	})
}
