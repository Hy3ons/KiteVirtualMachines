package cloudinituserdata

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed ubuntu-22.04-init.yaml
var ubuntu2204CloudInitTemplate string

type Ubuntu2204CloudInit struct {
	VmName    string
	Namespace string
	Id        string
	Password  string
}

// Render creates a cloud-init Secret object for an Ubuntu 22.04 virtual machine.
// The receiver provides VM name, namespace, login id, and password template values.
// The returned object is applied by the KiteVirtualMachine reconcile flow.
// This method uses an embedded template so the controller does not depend on source-tree files at runtime.
func (s *Ubuntu2204CloudInit) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("ubuntu-22.04-init.yaml", ubuntu2204CloudInitTemplate)
	return renderer.Render(s)
}
