package platformhttpredirect

import "testing"

func TestPlatformHTTPRedirectRenderRejectsEmptyHost(t *testing.T) {
	obj, err := (&PlatformHTTPRedirectData{Namespace: "kite"}).Render()
	if err == nil {
		t.Fatalf("expected empty host to fail, got %#v", obj)
	}
}

func TestPlatformHTTPRedirectRender(t *testing.T) {
	obj, err := (&PlatformHTTPRedirectData{Namespace: "kite", Host: "domain.com"}).Render()
	if err != nil {
		t.Fatalf("failed to render platform HTTP redirect ingress: %v", err)
	}

	if obj.GetName() != "kite-platform-http-redirect" {
		t.Fatalf("expected redirect ingress name, got %q", obj.GetName())
	}
	if obj.GetNamespace() != "kite" {
		t.Fatalf("expected kite namespace, got %q", obj.GetNamespace())
	}
	annotations := obj.GetAnnotations()
	if annotations["traefik.ingress.kubernetes.io/router.entrypoints"] != "web" {
		t.Fatalf("expected web entrypoint, got %#v", annotations)
	}
	if annotations["traefik.ingress.kubernetes.io/router.middlewares"] != "kite-kite-platform-https-redirect@kubernetescrd" {
		t.Fatalf("expected HTTPS redirect middleware annotation, got %#v", annotations)
	}
}
