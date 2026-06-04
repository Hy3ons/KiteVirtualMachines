package apps

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"kite/internal/kube"
)

// optionalResourceAvailable checks whether an optional Kubernetes resource is installed.
// clientManager provides the typed Kubernetes client used for API discovery.
// gvr identifies the optional resource that an informer wants to watch.
// The returned boolean is false when the CRD or APIService is not available in the cluster.
func optionalResourceAvailable(clientManager *kube.ClientManager, gvr schema.GroupVersionResource) (bool, error) {
	if clientManager == nil || clientManager.KubeClient == nil {
		return false, fmt.Errorf("Kubernetes discovery client is required")
	}

	resourceList, err := clientManager.KubeClient.Discovery().ServerResourcesForGroupVersion(gvr.GroupVersion().String())
	if err != nil {
		return false, nil
	}

	for _, resource := range resourceList.APIResources {
		if resource.Name == gvr.Resource {
			return true, nil
		}
	}

	return false, nil
}
