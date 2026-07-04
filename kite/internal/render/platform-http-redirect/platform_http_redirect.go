package platformhttpredirect

import (
	_ "embed"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/render"
)

//go:embed platform-http-redirect.yaml
var platformHTTPRedirectTemplate string

// PlatformHTTPRedirectData contains values for the Traefik HTTP redirect Ingress template.
// Namespace is where kite-api, kite-frontend, and the redirect middleware live.
// Host is required so HTTPS redirects cannot capture unrelated hostnames.
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
	if strings.TrimSpace(d.Host) == "" {
		return nil, fmt.Errorf("platform HTTP redirect host is required")
	}

	renderer := render.NewRendererFromTemplate("platform-http-redirect.yaml", platformHTTPRedirectTemplate)
	return renderer.Render(d)
}
