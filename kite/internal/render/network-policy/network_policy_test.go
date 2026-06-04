package networkpolicy

import "testing"

func TestNetworkPolicyRenderAllAllowsKiteAndCDINamespaces(t *testing.T) {
	objects, err := (&NetworkPolicyData{Namespace: "tenant-a"}).RenderAll()
	if err != nil {
		t.Fatalf("failed to render network policies: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("expected 2 network policies, got %d", len(objects))
	}

	rendered := objects[0].Object
	if !containsNamespaceSelector(rendered, "spec", "ingress", "from", "kite") {
		t.Fatal("expected ingress policy to allow kite namespace")
	}
	if !containsNamespaceSelector(rendered, "spec", "ingress", "from", "cdi") {
		t.Fatal("expected ingress policy to allow cdi namespace")
	}

	rendered = objects[1].Object
	if !containsNamespaceSelector(rendered, "spec", "egress", "to", "kite") {
		t.Fatal("expected egress policy to allow kite namespace")
	}
	if !containsNamespaceSelector(rendered, "spec", "egress", "to", "cdi") {
		t.Fatal("expected egress policy to allow cdi namespace")
	}
}

func containsNamespaceSelector(obj map[string]interface{}, listField string, ruleField string, peerField string, namespace string) bool {
	spec, ok := obj[listField].(map[string]interface{})
	if !ok {
		return false
	}

	rules, ok := spec[ruleField].([]interface{})
	if !ok {
		return false
	}

	for _, ruleValue := range rules {
		rule, ok := ruleValue.(map[string]interface{})
		if !ok {
			continue
		}
		peers, ok := rule[peerField].([]interface{})
		if !ok {
			continue
		}
		for _, peerValue := range peers {
			peer, ok := peerValue.(map[string]interface{})
			if !ok {
				continue
			}
			if namespaceSelectorName(peer) == namespace {
				return true
			}
		}
	}

	return false
}

func namespaceSelectorName(peer map[string]interface{}) string {
	selector, ok := peer["namespaceSelector"].(map[string]interface{})
	if !ok {
		return ""
	}
	labels, ok := selector["matchLabels"].(map[string]interface{})
	if !ok {
		return ""
	}
	name, ok := labels["kubernetes.io/metadata.name"].(string)
	if !ok {
		return ""
	}
	return name
}
