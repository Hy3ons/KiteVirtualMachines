package kubevirtmachine

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type KubevirtMachineData struct {
	VmName    string
	Namespace string
	Memory string
	CPU string
}

func (s *KubevirtMachineData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/kubevirt-machine/kubevirt-machine.yaml")
	return renderer.Render(s)
}
