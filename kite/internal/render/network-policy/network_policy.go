package networkpolicy

import (
	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type NetworkPolicyData struct {
	Namespace string
}

func (s *NetworkPolicyData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRenderer("kite/internal/render/network-policy/network-policy.yaml")
	return renderer.Render(s)
}
