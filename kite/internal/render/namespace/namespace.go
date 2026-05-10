package namespace

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed namespace.yaml
var namespaceTemplate string

type NamespaceData struct {
	Namespace string
}

// Render creates a Kubernetes Namespace object from NamespaceData.
// The receiver provides the metadata.name value for the namespace.
// The returned object is applied by the KiteUser controller reconcile flow.
// This method uses an embedded template so the controller does not depend on source-tree files at runtime.
func (s *NamespaceData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("namespace.yaml", namespaceTemplate)
	return renderer.Render(s)
}
