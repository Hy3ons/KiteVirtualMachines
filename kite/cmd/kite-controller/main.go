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
)

func main() {
	clientManager, err := getDynamicClient()

	if err != nil {
		log.Fatalf("클러스터에 연결을 실패하여, 종료합니다.")
	}

	gvr := schema.GroupVersionResource{
		Group: "anacnu.com",
		Version: "v1",
		Resource: "kitevirtualmachines",
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, time.Second * 30)
	informer := factory.ForResource(gvr).Informer()

	//eventChan := make(chan string)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func (obj interface{}) {
			resource, ok := obj.(*unstructured.Unstructured)

			var kite kite.KiteVirtualMachine

			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &kite); err != nil {
				fmt.Printf("Error : %v\n", err)
				return
			}

			if !ok {
				log.Fatalf("Type Casting Error : It is not a unstructured Type...")
			}

			log.Printf("AddFunc %s\n", resource.GetName())

			//cpu := resource.Object["cpu"].(int)
			//log.Printf("New Resource!!. Cpu is %d\n", cpu)
		},

		UpdateFunc: func (oldObj interface{}, newObj interface{}) {
			oldResource, ok := oldObj.(metav1.Object)
			_, ok = newObj.(metav1.Object)

			//var IsSameVersion bool = oldResource.GetResourceVersion() == newResource.GetResourceVersion()

			if !ok {
				log.Fatalf("Type Casting Error : It is not a unstructured Type...")
			}

			log.Printf("UpdateFunc %s\n", oldResource.GetName())

			//cpu := resource.Object["cpu"].(int)
			//log.Printf("New Resource!!. Cpu is %d\n", cpu)
		},

		DeleteFunc: func (obj interface{}) {
			resource, ok := obj.(*unstructured.Unstructured)

			if !ok {
				log.Printf("Kite Resource가, Unstructured 타입이 아닙니다.")
				return
			}

			var kite kite.KiteVirtualMachine

			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &kite); err != nil {


				return
			}
			

			

			
			log.Printf("Something went gone")
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	go factory.Start(stopCh)
	time.Sleep(time.Minute * 3)
}