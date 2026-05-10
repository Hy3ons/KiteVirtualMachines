package quotapolicy

import (
	_ "embed"

	"kite/internal/render"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:embed quota-policy.yaml
var quotaPolicyTemplate string

type QuotaPolicyData struct {
	VmName    string
	Namespace string
}

// Render creates the first Kubernetes quota object from QuotaPolicyData.
// VmName is used as the shared quota resource name.
// Namespace is the user namespace where the quota object should be created.
// The returned object is used by callers that only need the first quota document.
func (s *QuotaPolicyData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("quota-policy.yaml", quotaPolicyTemplate)
	return renderer.Render(s)
}

// RenderAll renders every quota policy document in the quota policy template.
// VmName is used as the shared ResourceQuota and LimitRange name.
// Namespace is the user namespace where those resources should be applied.
// The returned objects are applied by the KiteUser controller reconcile flow.
func (s *QuotaPolicyData) RenderAll() ([]*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("quota-policy.yaml", quotaPolicyTemplate)
	return renderer.RenderAll(s)
}
