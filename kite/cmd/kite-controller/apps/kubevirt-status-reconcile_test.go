package apps

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	kite "kite/api/v1"
)

func TestKiteStatusKeepsProvisioningWhenDesiredOnAndKubeVirtStopped(t *testing.T) {
	kiteVM := newKubeVirtStatusTestKiteVM("On")
	kubeVirtVM := newKubeVirtStatusTestVM("Always", kubeVirtStatusStopped)

	phase, powerState, conditionStatus, message := kiteStatusFromKubeVirtVirtualMachine(kiteVM, kubeVirtVM)

	if phase != kiteVMPhaseProvisioning {
		t.Fatalf("expected phase %q, got %q", kiteVMPhaseProvisioning, phase)
	}
	if powerState != "Off" {
		t.Fatalf("expected current power state Off, got %q", powerState)
	}
	if conditionStatus != metav1.ConditionFalse {
		t.Fatalf("expected condition status False, got %s", conditionStatus)
	}
	if message != "VM start is pending while KubeVirt reports Stopped" {
		t.Fatalf("unexpected message %q", message)
	}
}

func TestKiteStatusStopsWhenDesiredOffAndKubeVirtStopped(t *testing.T) {
	kiteVM := newKubeVirtStatusTestKiteVM("Off")
	kubeVirtVM := newKubeVirtStatusTestVM("Halted", kubeVirtStatusStopped)

	phase, powerState, conditionStatus, message := kiteStatusFromKubeVirtVirtualMachine(kiteVM, kubeVirtVM)

	if phase != kiteVMPhaseStopped {
		t.Fatalf("expected phase %q, got %q", kiteVMPhaseStopped, phase)
	}
	if powerState != "Off" {
		t.Fatalf("expected current power state Off, got %q", powerState)
	}
	if conditionStatus != metav1.ConditionTrue {
		t.Fatalf("expected condition status True, got %s", conditionStatus)
	}
	if message != "KubeVirt VirtualMachine is stopped" {
		t.Fatalf("unexpected message %q", message)
	}
}

func newKubeVirtStatusTestKiteVM(powerState string) *kite.KiteVirtualMachine {
	return &kite.KiteVirtualMachine{
		Spec: kite.KiteVirtualMachineSpec{
			PowerState: powerState,
		},
	}
}

func newKubeVirtStatusTestVM(runStrategy string, printableStatus string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"spec": map[string]any{
				"runStrategy": runStrategy,
			},
			"status": map[string]any{
				"printableStatus": printableStatus,
			},
		},
	}
}
