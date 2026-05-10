package networkpolicy

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed network-policy.yaml
var networkPolicyTemplate string

type NetworkPolicyData struct {
	Namespace string
}

// Render creates the first Kubernetes NetworkPolicy object from NetworkPolicyData.
// The receiver provides the namespace where the policy should be created.
// The returned object is used by callers that only need the first policy document.
// This method uses an embedded template so the controller does not depend on source-tree files at runtime.
func (s *NetworkPolicyData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("network-policy.yaml", networkPolicyTemplate)
	return renderer.Render(s)
}

// RenderAll renders every NetworkPolicy document in the network policy template.
// The receiver provides the namespace that each policy should be created in.
// The returned objects are applied by the KiteUser controller reconcile flow.
func (s *NetworkPolicyData) RenderAll() ([]*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("network-policy.yaml", networkPolicyTemplate)
	return renderer.RenderAll(s)
}
