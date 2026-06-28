package platformhttpredirect

import (
	_ "embed"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/render"
)

//go:embed platform-http-redirect.yaml
var platformHTTPRedirectTemplate string

// PlatformHTTPRedirectData contains values for the Traefik HTTP redirect Ingress template.
// Namespace is where kite-api, kite-frontend, and the redirect middleware live.
// Host is optional; when empty the redirect Ingress accepts requests without host matching.
// This renderer is used by kite-controller when platform HTTPS enforcement is enabled.
type PlatformHTTPRedirectData struct {
	Namespace string
	Host      string
}

// Render creates the platform HTTP redirect Ingress object.
// The receiver provides namespace and optional host template values.
// The returned Ingress attaches the HTTPS redirect middleware to the web entrypoint.
// This method uses an embedded template so the controller works from a container image.
func (d *PlatformHTTPRedirectData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("platform-http-redirect.yaml", platformHTTPRedirectTemplate)
	return renderer.Render(d)
}
