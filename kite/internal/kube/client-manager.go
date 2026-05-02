package kube

import (
	"fmt"
	"log"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetClientManager creates Kubernetes typed and dynamic clients from cluster configuration.
// It first tries in-cluster configuration, then falls back to the local kubeconfig for development.
// The returned ClientManager is used by controller startup and integration tests that need real Kubernetes clients.
func GetClientManager() (*ClientManager, error) {
	config, err := rest.InClusterConfig()

	if err != nil { // 일단은 로컬로 재시도.
		log.Printf("로컬로 Cluster 연결을 재시도 합니다. \n%v\n", err.Error())
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	}

	if err != nil { // 에러가 발생하면, Error Log를 뱉고, 프로그램 종료.
		return nil, fmt.Errorf("연결할 클러스터를 찾지 못했습니다. \n%v\n", err.Error())
	}

	dynClient, err := dynamic.NewForConfig(config)

	if err != nil {
		return nil, fmt.Errorf("Dynamic Client 생성 실패 : %v\n", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, fmt.Errorf("Kube Client 생성 실패 : %v\n", err.Error())
	}

	return &ClientManager{
		DynamicClient: dynClient,
		KubeClient:    kubeClient,
	}, nil
}
