package platformhttpsredirect

import (
	_ "embed"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/render"
)

//go:embed platform-https-redirect.yaml
var platformHTTPSRedirectTemplate string

// PlatformHTTPSRedirectData contains values for the Traefik redirect middleware template.
// Namespace is where the platform Ingress lives.
// This renderer is used by kite-controller when platform HTTPS enforcement is enabled.
type PlatformHTTPSRedirectData struct {
	Namespace string
}

// Render creates the Traefik Middleware object for platform HTTPS redirects.
// The receiver provides the namespace template value.
// The returned object redirects HTTP requests to HTTPS permanently.
// This method uses an embedded template so the controller works from a container image.
func (d *PlatformHTTPSRedirectData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("platform-https-redirect.yaml", platformHTTPSRedirectTemplate)
	return renderer.Render(d)
}
