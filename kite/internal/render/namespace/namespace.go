package namespace

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type NamespaceData struct {
	Namespace string
}

func (s *NamespaceData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/namespace/namespace.yaml")
	return renderer.Render(s)
}
