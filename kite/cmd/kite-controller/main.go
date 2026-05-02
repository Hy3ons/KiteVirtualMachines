package main

import (
	"fmt"
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

func main() {
	clientManager, err := kube.GetClientManager()

	if err != nil {
		log.Fatalf("클러스터에 연결을 실패하여, 종료합니다.")
	}

	gvr := schema.GroupVersionResource{
		Group:    "anacnu.com",
		Version:  "v1",
		Resource: "kitevirtualmachines",
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, time.Second*30)
	informer := factory.ForResource(gvr).Informer()

	//eventChan := make(chan string)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			resource, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Printf("KiteVirtualMachine add event object is not unstructured")
				return
			}

			var vm kite.KiteVirtualMachine
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
				fmt.Printf("Error : %v\n", err)
				return
			}

			log.Printf("AddFunc %s\n", resource.GetName())

			//cpu := resource.Object["cpu"].(int)
			//log.Printf("New Resource!!. Cpu is %d\n", cpu)
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

			if oldResource.GetResourceVersion() == newResource.GetResourceVersion() {
				return
			}

			log.Printf("UpdateFunc %s\n", oldResource.GetName())

			//cpu := resource.Object["cpu"].(int)
			//log.Printf("New Resource!!. Cpu is %d\n", cpu)
		},

		DeleteFunc: func(obj interface{}) {
			resource, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Printf("Kite Resource가, Unstructured 타입이 아닙니다.")
				return
			}

			var vm kite.KiteVirtualMachine
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
				fmt.Printf("Error : %v\n", err)
				return
			}

			log.Printf("DeleteFunc %s\n", resource.GetName())
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	go factory.Start(stopCh)
	time.Sleep(time.Minute * 3)
}
