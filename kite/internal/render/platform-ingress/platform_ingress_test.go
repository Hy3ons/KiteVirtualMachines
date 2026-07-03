package platformingress

import "testing"

func TestPlatformIngressRenderWithoutHost(t *testing.T) {
	obj, err := (&PlatformIngressData{Namespace: "kite"}).Render()
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
	if _, exists := rule["host"]; exists {
		t.Fatalf("expected no host for default platform ingress, got %#v", rule["host"])
	}
}

func TestPlatformIngressRenderWithHost(t *testing.T) {
	obj, err := (&PlatformIngressData{Namespace: "kite", Host: "domain.com"}).Render()
	if err != nil {
		t.Fatalf("failed to render platform ingress: %v", err)
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
