package platformhttpsredirect

import "testing"

func TestPlatformHTTPSRedirectRender(t *testing.T) {
	obj, err := (&PlatformHTTPSRedirectData{Namespace: "kite"}).Render()
	if err != nil {
		t.Fatalf("failed to render platform HTTPS redirect middleware: %v", err)
	}

	if obj.GetAPIVersion() != "traefik.io/v1alpha1" {
		t.Fatalf("expected Traefik middleware apiVersion, got %q", obj.GetAPIVersion())
	}
	if obj.GetKind() != "Middleware" {
		t.Fatalf("expected Middleware kind, got %q", obj.GetKind())
	}
	if obj.GetName() != "kite-platform-https-redirect" {
		t.Fatalf("expected redirect middleware name, got %q", obj.GetName())
	}
	if obj.GetNamespace() != "kite" {
		t.Fatalf("expected kite namespace, got %q", obj.GetNamespace())
	}
}
