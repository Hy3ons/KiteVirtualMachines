package datavolume

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed data-volume.yaml
var dataVolumeTemplate string

// VmName should be ubuntu-22.04.

type VmName string

const (
	Ubuntu2204 VmName = "ubuntu-22.04"
)

type DataVolumeData struct {
	VmName    VmName
	Namespace string
	VmImage   string
	Storage   string
}

// Render creates a KubeVirt DataVolume object from DataVolumeData.
// The receiver provides VM name, namespace, image source, and storage size template values.
// The returned object is applied by the KiteVirtualMachine reconcile flow.
// This method uses an embedded template so the controller does not depend on source-tree files at runtime.
func (s *DataVolumeData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("data-volume.yaml", dataVolumeTemplate)
	return renderer.Render(s)
}
