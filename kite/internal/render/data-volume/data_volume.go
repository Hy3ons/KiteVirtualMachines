package datavolume

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//VmName should be ubuntu-22.04

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

func (s *DataVolumeData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/data-volume/data-volume.yaml")
	return renderer.Render(s)
}
