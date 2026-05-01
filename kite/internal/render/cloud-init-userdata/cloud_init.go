package cloudinituserdata

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Ubuntu2204CloudInit struct {
	VmName string
	Namespace string
	Id string
	Password string
}

func (s *Ubuntu2204CloudInit) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/cloud-init-userdata/ubuntu-22.04-init.yaml")
	return renderer.Render(s)
}
