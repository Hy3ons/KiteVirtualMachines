package vmservice

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed vm-service.yaml
var vmServiceTemplate string

type ServiceData struct {
	VmName    string
	Namespace string
	NodePort  string
}

// Render creates a Kubernetes Service object from ServiceData.
// The receiver provides VM name, namespace, and node port template values.
// The returned object is applied by the KiteVirtualMachine reconcile flow.
// This method uses an embedded template so the controller does not depend on source-tree files at runtime.
func (s *ServiceData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("vm-service.yaml", vmServiceTemplate)
	return renderer.Render(s)
}
