package datavolume

import "testing"

func TestDataVolumeRenderUsesGoldenPVCSource(t *testing.T) {
	obj, err := (&DataVolumeData{
		VmName:    "vm-a",
		Namespace: "user-a",
		VmImage:   Ubuntu2204,
		Storage:   "25Gi",
	}).Render()
	if err != nil {
		t.Fatalf("failed to render DataVolume: %v", err)
	}

	sourceName, _, _ := unstructuredNestedString(obj.Object, "spec", "source", "pvc", "name")
	if sourceName != "ubuntu-22.04" {
		t.Fatalf("expected Ubuntu golden PVC source name, got %q", sourceName)
	}
	sourceNamespace, _, _ := unstructuredNestedString(obj.Object, "spec", "source", "pvc", "namespace")
	if sourceNamespace != "kite" {
		t.Fatalf("expected golden PVC source namespace kite, got %q", sourceNamespace)
	}
	if _, found, _ := unstructuredNestedString(obj.Object, "spec", "pvc", "storageClassName"); found {
		t.Fatal("expected storageClassName to be omitted so the cluster default is used")
	}
}

// unstructuredNestedString reads a nested string without leaking Kubernetes helpers into assertions.
// object is an unstructured Kubernetes object map.
// fields identifies the nested path to read.
// The returned values match unstructured.NestedString.
func unstructuredNestedString(object map[string]any, fields ...string) (string, bool, error) {
	current := any(object)
	for _, field := range fields {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return "", false, nil
		}
		next, ok := currentMap[field]
		if !ok {
			return "", false, nil
		}
		current = next
	}

	value, ok := current.(string)
	return value, ok, nil
}
