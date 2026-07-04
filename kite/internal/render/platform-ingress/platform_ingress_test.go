package platformingress

import "testing"

func TestPlatformIngressRenderRejectsEmptyHost(t *testing.T) {
	obj, err := (&PlatformIngressData{Namespace: "kite"}).Render()
	if err == nil {
		t.Fatalf("expected empty host to fail, got %#v", obj)
	}
}

func TestPlatformIngressRenderWithHost(t *testing.T) {
	obj, err := (&PlatformIngressData{Namespace: "kite", Host: "domain.com"}).Render()
	if err != nil {
		t.Fatalf("failed to render platform ingress: %v", err)
	}
	if obj.GetName() != "kite-platform" {
		t.Fatalf("expected kite-platform ingress name, got %q", obj.GetName())
	}
	if obj.GetNamespace() != "kite" {
		t.Fatalf("expected kite namespace, got %q", obj.GetNamespace())
	}

	rules, ok := obj.Object["spec"].(map[string]any)["rules"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("expected one ingress rule, got %#v", obj.Object["spec"])
	}
	rule, ok := rules[0].(map[string]any)
	if !ok {
		t.Fatalf("expected rule object, got %#v", rules[0])
	}
	if rule["host"] != "domain.com" {
		t.Fatalf("expected domain.com host, got %#v", rule["host"])
	}
}

func TestPlatformIngressRenderWithTLSSecret(t *testing.T) {
	obj, err := (&PlatformIngressData{Namespace: "kite", Host: "domain.com", TLSSecretName: "global-tls-secret"}).Render()
	if err != nil {
		t.Fatalf("failed to render platform ingress: %v", err)
	}

	tlsEntries, ok := obj.Object["spec"].(map[string]any)["tls"].([]any)
	if !ok || len(tlsEntries) != 1 {
		t.Fatalf("expected one tls entry, got %#v", obj.Object["spec"])
	}
	tlsEntry, ok := tlsEntries[0].(map[string]any)
	if !ok {
		t.Fatalf("expected tls object, got %#v", tlsEntries[0])
	}
	if tlsEntry["secretName"] != "global-tls-secret" {
		t.Fatalf("expected global TLS secret, got %#v", tlsEntry["secretName"])
	}
}

func TestPlatformIngressRenderWithForceHTTPS(t *testing.T) {
	obj, err := (&PlatformIngressData{Namespace: "kite", Host: "domain.com", ForceHTTPS: true}).Render()
	if err != nil {
		t.Fatalf("failed to render platform ingress: %v", err)
	}

	annotations := obj.GetAnnotations()
	if annotations["traefik.ingress.kubernetes.io/router.entrypoints"] != "websecure" {
		t.Fatalf("expected websecure entrypoint, got %#v", annotations)
	}
	if annotations["traefik.ingress.kubernetes.io/router.tls"] != "true" {
		t.Fatalf("expected TLS annotation, got %#v", annotations)
	}
}
