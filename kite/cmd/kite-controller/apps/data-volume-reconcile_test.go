package apps

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

// TestUpdateKiteVirtualMachineDataVolumeStatusPreservesRuntimePhase verifies disk progress does not lower VM state.
// t is the Go test handle used for assertions.
// The test models a DataVolume resync after KubeVirt already reported a stopped VM.
func TestUpdateKiteVirtualMachineDataVolumeStatusPreservesRuntimePhase(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	name := "vm-a"
	vm := newMachineReconcileKiteVirtualMachineSpec(namespace, name)
	setMachineReconcileStatus(t, vm, kiteVMPhaseStopped, "Off", 1, metav1.ConditionTrue, kiteVMReasonReconciled, "KubeVirt VirtualMachine is stopped")
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(), vm)

	typedVM, err := kiteVirtualMachineFromEventObject(vm)
	if err != nil {
		t.Fatalf("failed to convert test VM: %v", err)
	}
	if err := updateKiteVirtualMachineDataVolumeStatus(ctx, client, typedVM, dataVolumePhaseSucceeded, "100.0%", "Import Complete", kiteVMPhaseProvisioning, metav1.ConditionFalse, kiteVMReasonDataVolumeReady); err != nil {
		t.Fatalf("updateKiteVirtualMachineDataVolumeStatus returned error: %v", err)
	}

	current, err := client.Resource(kiteVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read updated VM: %v", err)
	}
	phase, _, _ := unstructured.NestedString(current.Object, "status", "phase")
	message := firstStatusConditionMessage(current)
	dataVolumePhase, _, _ := unstructured.NestedString(current.Object, "status", "dataVolumePhase")
	if phase != kiteVMPhaseStopped {
		t.Fatalf("expected stable phase %q to be preserved, got %q", kiteVMPhaseStopped, phase)
	}
	if message != "KubeVirt VirtualMachine is stopped" {
		t.Fatalf("expected stable condition message to be preserved, got %q", message)
	}
	if dataVolumePhase != dataVolumePhaseSucceeded {
		t.Fatalf("expected DataVolume phase %q to be recorded, got %q", dataVolumePhaseSucceeded, dataVolumePhase)
	}
}
