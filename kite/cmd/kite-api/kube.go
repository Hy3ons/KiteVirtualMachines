package main

import (
	"fmt"
	"log"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// getClusterConfig loads Kubernetes REST config for kite-api.
// It first uses in-cluster service account config, then local kubeconfig for development.
// The returned config is shared by dynamic CRD clients and KubeVirt console proxy code.
// This function is used during kite-api startup before the Gin router is created.
func getClusterConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("retry cluster connection with local kubeconfig: %v\n", err)
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find cluster config: %w", err)
	}

	return config, nil
}

// getDynamicClient creates a Kubernetes dynamic client from REST config.
// config is the cluster connection settings returned by getClusterConfig.
// The returned client reads and writes Kite CRDs and Kubernetes runtime config objects.
// This function is used by kite-api startup after config discovery succeeds.
func getDynamicClient(config *rest.Config) (dynamic.Interface, error) {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return client, nil
}
