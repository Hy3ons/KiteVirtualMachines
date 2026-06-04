package kubevirtmachine

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed kubevirt-machine.yaml
var kubevirtMachineTemplate string

type KubevirtMachineData struct {
	VmName      string
	Namespace   string
	Memory      string
	CPU         string
	RunStrategy string
}

// Render creates a KubeVirt VirtualMachine object from KubevirtMachineData.
// The receiver provides VM name, namespace, memory, and CPU template values.
// The returned object is applied by the KiteVirtualMachine reconcile flow.
// This method uses an embedded template so the controller does not depend on source-tree files at runtime.
func (s *KubevirtMachineData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("kubevirt-machine.yaml", kubevirtMachineTemplate)
	return renderer.Render(s)
}
