package vmservice

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ServiceData struct {
	VmName string
	Namespace string
	NodePort string
}

func (s *ServiceData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/service/vm-service.yaml")
	return renderer.Render(s)
}
	

