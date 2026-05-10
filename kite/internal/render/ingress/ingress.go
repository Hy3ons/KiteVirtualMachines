package ingress

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed ingress.yaml
var ingressTemplate string

type IngressData struct {
	VmName     string
	Namespace  string
	DomainName string
}

// Render creates a Kubernetes Ingress object from IngressData.
// The receiver provides VM name, namespace, and domain name template values.
// The returned object is applied by the KiteVirtualMachine reconcile flow.
// This method uses an embedded template so the controller does not depend on source-tree files at runtime.
func (s *IngressData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("ingress.yaml", ingressTemplate)
	return renderer.Render(s)
}
