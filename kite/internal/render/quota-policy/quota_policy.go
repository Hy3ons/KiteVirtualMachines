package quotapolicy

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type QuotaPolicyData struct {
	VmName    string
	Namespace string
}

func (s *QuotaPolicyData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/quota-policy/quota-policy.yaml")
	return renderer.Render(s)
}
