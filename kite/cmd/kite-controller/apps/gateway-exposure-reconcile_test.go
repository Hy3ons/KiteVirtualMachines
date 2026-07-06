package apps

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/platform"
)

// TestGatewayBlockedStatusRejectsMissingExternalPort verifies external exposure requires an explicit port.
// t is the Go test handle used for assertions.
// The controller must keep kite-gateway-external absent when operator desired state is incomplete.
func TestGatewayBlockedStatusRejectsMissingExternalPort(t *testing.T) {
	status, blocked := gatewayBlockedStatus(platform.SSHGatewayDesired{ExternalEnabled: true})
	if !blocked {
		t.Fatal("expected missing external port to block gateway exposure")
	}
	if status.Reason != platform.SSHGatewayReasonMissingExternalPort {
		t.Fatalf("unexpected reason %q", status.Reason)
	}
}

// TestGatewayBlockedStatusRejectsMissingHostFallbackPort verifies host fallback needs an explicit host sshd port.
// t is the Go test handle used for assertions.
// This protects host access because Kite no longer guesses where host sshd is listening.
func TestGatewayBlockedStatusRejectsMissingHostFallbackPort(t *testing.T) {
	status, blocked := gatewayBlockedStatus(platform.SSHGatewayDesired{
		ExternalEnabled:     true,
		ExternalPort:        "40022",
		HostFallbackEnabled: true,
	})
	if !blocked {
		t.Fatal("expected missing host sshd port to block gateway exposure")
	}
	if status.Reason != platform.SSHGatewayReasonMissingHostFallbackPort {
		t.Fatalf("unexpected reason %q", status.Reason)
	}
}

// TestGatewayBlockedStatusRejectsPortConflict verifies VM gateway and host sshd ports cannot match.
// t is the Go test handle used for assertions.
// The controller must not create a state where VM SSH exposure can mask direct host access.
func TestGatewayBlockedStatusRejectsPortConflict(t *testing.T) {
	status, blocked := gatewayBlockedStatus(platform.SSHGatewayDesired{
		ExternalEnabled:     true,
		ExternalPort:        "22",
		HostFallbackEnabled: true,
		HostSshdPort:        "22",
	})
	if !blocked {
		t.Fatal("expected port conflict to block gateway exposure")
	}
	if status.Reason != platform.SSHGatewayReasonPortConflict {
		t.Fatalf("unexpected reason %q", status.Reason)
	}
}

// TestGatewayBlockedStatusAllowsSafeDesiredState verifies complete non-conflicting settings pass validation.
// t is the Go test handle used for assertions.
// A nil blocked state allows the reconciler to apply kite-gateway-external.
func TestGatewayBlockedStatusAllowsSafeDesiredState(t *testing.T) {
	_, blocked := gatewayBlockedStatus(platform.SSHGatewayDesired{
		ExternalEnabled:     true,
		ExternalPort:        "40022",
		HostFallbackEnabled: true,
		HostSshdPort:        "22",
	})
	if blocked {
		t.Fatal("expected non-conflicting gateway settings to pass")
	}
}

// TestExternalGatewayServiceStatusReportsPendingLoadBalancer verifies Service apply is not treated as ready too early.
// t is the Go test handle used for assertions.
// This gives admins a clear status when a requested port cannot be assigned by the cluster LoadBalancer.
func TestExternalGatewayServiceStatusReportsPendingLoadBalancer(t *testing.T) {
	service := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name":      kiteGatewayExternalServiceName,
				"namespace": "kite",
			},
			"spec": map[string]any{
				"type": "LoadBalancer",
			},
		},
	}

	status := externalGatewayServiceStatusFromObject(service, "40022", "$(KITE_NODE_IP):22")
	if status.Phase != platform.SSHGatewayPhaseReconciling {
		t.Fatalf("unexpected phase %q", status.Phase)
	}
	if status.Reason != platform.SSHGatewayReasonServicePending {
		t.Fatalf("unexpected reason %q", status.Reason)
	}
	if status.ObservedExternalPort != "40022" {
		t.Fatalf("unexpected observed external port %q", status.ObservedExternalPort)
	}
}

// TestExternalGatewayServiceStatusReportsReadyLoadBalancer verifies observed ingress promotes the status to Ready.
// t is the Go test handle used for assertions.
// This is used by the Service informer path after the cluster LoadBalancer accepts the desired port.
func TestExternalGatewayServiceStatusReportsReadyLoadBalancer(t *testing.T) {
	service := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name":      kiteGatewayExternalServiceName,
				"namespace": "kite",
			},
			"status": map[string]any{
				"loadBalancer": map[string]any{
					"ingress": []any{
						map[string]any{"ip": "192.0.2.10"},
					},
				},
			},
		},
	}

	status := externalGatewayServiceStatusFromObject(service, "40022", "")
	if status.Phase != platform.SSHGatewayPhaseReady {
		t.Fatalf("unexpected phase %q", status.Phase)
	}
	if status.Reason != platform.SSHGatewayReasonServiceApplied {
		t.Fatalf("unexpected reason %q", status.Reason)
	}
}
