package main

import (
	"log"

	"kite/cmd/kite-controller/apps"
	"kite/internal/kube"
)

func main() {
	clientManager, err := kube.GetClientManager()

	if err != nil {
		log.Fatalf("클러스터에 연결을 실패하여, 종료합니다.")
	}

	stopCh := make(chan struct{})

	go apps.RunKiteUserReconciler(clientManager, stopCh)
	go apps.RunKiteNamespaceReconciler(clientManager, stopCh)
	go apps.RunKiteVirtualMachineReconciler(clientManager, stopCh)

	select {}
}
