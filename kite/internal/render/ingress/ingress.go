package ingress

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type IngressData struct {
	VmName     string
	Namespace  string
	DomainName string
}

func (s *IngressData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/ingress/ingress.yaml")
	return renderer.Render(s)
}
