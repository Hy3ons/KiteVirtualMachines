package platformingress

import (
	_ "embed"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/render"
)

//go:embed platform-ingress.yaml
var platformIngressTemplate string

// PlatformIngressData contains values for the Kite platform Ingress template.
// Namespace is where kite-api and kite-frontend Services live.
// Host is optional; an empty value renders the default hostless HTTP route used after install.
// ForceHTTPS controls whether Traefik redirects platform HTTP traffic to HTTPS.
// TLSSecretName is the optional namespace-local TLS Secret referenced by spec.tls.
// This renderer is used by kite-controller to expose frontend and API through one Ingress.
type PlatformIngressData struct {
	Namespace     string
	Host          string
	ForceHTTPS    bool
	TLSSecretName string
}

// Render creates the Kite platform Ingress object.
// The receiver provides namespace and optional host template values.
// The returned object routes /api to kite-api and all other paths to kite-frontend.
// This method uses an embedded template so the controller works from a container image.
func (d *PlatformIngressData) Render() (*unstructured.Unstructured, error) {
	renderer := render.NewRendererFromTemplate("platform-ingress.yaml", platformIngressTemplate)
	return renderer.Render(d)
}
