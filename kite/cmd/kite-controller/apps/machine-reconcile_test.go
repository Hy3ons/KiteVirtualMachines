package apps

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

// TestReconcileKiteVirtualMachineDeleteIntentDeletesKubeVirtFirst verifies delete intent cleanup order.
// t is the Go test handle used for assertions.
// The test is used by the controller package to keep KubeVirt cleanup before Kite CRD cleanup.
func TestReconcileKiteVirtualMachineDeleteIntentDeletesKubeVirtFirst(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	name := "vm-a"
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcileKiteVirtualMachine(namespace, name, true),
		newMachineReconcileKubeVirtVirtualMachine(namespace, name),
	)

	if err := ReconcileKiteVirtualMachine(ctx, client, newMachineReconcileKiteVirtualMachine(namespace, name, true)); err != nil {
		t.Fatalf("ReconcileKiteVirtualMachine returned error: %v", err)
	}

	if _, err := client.Resource(kubeVirtVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected KubeVirt VirtualMachine to be deleted first, got error %v", err)
	}
	if _, err := client.Resource(kiteVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
		t.Fatalf("expected KiteVirtualMachine CRD to remain while KubeVirt deletion is observed, got %v", err)
	}
}

// TestReconcileKiteVirtualMachineDeleteIntentDeletesCRDWhenKubeVirtMissing verifies final CRD cleanup.
// t is the Go test handle used for assertions.
// The test is used by the controller package to avoid orphan KiteVirtualMachine CRDs.
func TestReconcileKiteVirtualMachineDeleteIntentDeletesCRDWhenKubeVirtMissing(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	name := "vm-a"
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcileKiteVirtualMachine(namespace, name, true),
	)

	if err := ReconcileKiteVirtualMachine(ctx, client, newMachineReconcileKiteVirtualMachine(namespace, name, true)); err != nil {
		t.Fatalf("ReconcileKiteVirtualMachine returned error: %v", err)
	}

	if _, err := client.Resource(kiteVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected KiteVirtualMachine CRD to be deleted when KubeVirt VM is missing, got error %v", err)
	}
}

// TestReconcileKiteVirtualMachineFailsClearlyWhenDataVolumeAPIMissing verifies CDI dependency errors.
// t is the Go test handle used for assertions.
// The test is used by the controller package to keep missing CDI failures readable in VM status.
func TestReconcileKiteVirtualMachineFailsClearlyWhenDataVolumeAPIMissing(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	name := "vm-a"
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcileKiteVirtualMachineSpec(namespace, name),
	)

	err := ReconcileKiteVirtualMachine(ctx, client, newMachineReconcileKiteVirtualMachineSpec(namespace, name))
	if err == nil {
		t.Fatal("expected reconcile to fail when DataVolume API is missing")
	}

	current, getErr := client.Resource(kiteVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if getErr != nil {
		t.Fatalf("expected KiteVirtualMachine to remain with failed status, got %v", getErr)
	}

	phase, _, _ := unstructured.NestedString(current.Object, "status", "phase")
	message := firstStatusConditionMessage(current)
	if phase != kiteVMPhaseFailed {
		t.Fatalf("expected phase %q, got %q", kiteVMPhaseFailed, phase)
	}
	if message != "DataVolume resource API is not installed in this cluster; install CDI before creating KiteVirtualMachine disks" {
		t.Fatalf("expected missing CDI status message, got %q", message)
	}
}

// firstStatusConditionMessage returns the first condition message from an unstructured status.
// obj is a KiteVirtualMachine object with a status.conditions slice.
// The returned string is empty when no message exists.
// This helper is used by reconcile tests that inspect CRD status output.
func firstStatusConditionMessage(obj *unstructured.Unstructured) string {
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if len(conditions) == 0 {
		return ""
	}

	condition, ok := conditions[0].(map[string]any)
	if !ok {
		return ""
	}

	message, _ := condition["message"].(string)
	return message
}

// machineReconcileListKinds returns fake dynamic client list kind mappings.
// The returned map lets the fake client list Kite and KubeVirt VM resources if a test needs it.
// This helper is used by machine reconcile unit tests.
func machineReconcileListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		kiteVirtualMachineGVR:     "KiteVirtualMachineList",
		kubeVirtVirtualMachineGVR: "VirtualMachineList",
		secretGVR:                 "SecretList",
	}
}

// newMachineReconcileKiteVirtualMachine creates an unstructured KiteVirtualMachine test object.
// namespace is metadata.namespace for the namespaced CRD.
// name is metadata.name for both Kite and KubeVirt VM resources.
// deleteIntent controls spec.delete in the returned object.
func newMachineReconcileKiteVirtualMachine(namespace string, name string, deleteIntent bool) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "anacnu.com/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"delete": deleteIntent,
			},
		},
	}
}

// newMachineReconcileKiteVirtualMachineSpec creates a valid non-delete KiteVirtualMachine test object.
// namespace is metadata.namespace for the namespaced CRD.
// name is metadata.name for both Kite and KubeVirt VM resources.
// The returned object has enough spec fields to reach dependency apply.
func newMachineReconcileKiteVirtualMachineSpec(namespace string, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "anacnu.com/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":       name,
				"namespace":  namespace,
				"generation": int64(1),
			},
			"spec": map[string]any{
				"cpu":         int64(2),
				"memory":      "4Gi",
				"image":       "ubuntu-22.04",
				"disk":        "25Gi",
				"powerState":  "Off",
				"sshId":       "ubuntu",
				"sshPassword": "password",
			},
		},
	}
}

// newMachineReconcileKubeVirtVirtualMachine creates an unstructured KubeVirt VirtualMachine test object.
// namespace is metadata.namespace for the KubeVirt VM.
// name is metadata.name and matches the KiteVirtualMachine name in these tests.
// The returned object is stored in the fake dynamic client.
func newMachineReconcileKubeVirtVirtualMachine(namespace string, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}
