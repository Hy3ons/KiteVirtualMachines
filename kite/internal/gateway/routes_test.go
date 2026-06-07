package gateway

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kite/internal/auth"
)

const routeTestPasswordSalt = "route-test-salt"

// TestRouteFromKiteVirtualMachineUsesStatusNames verifies route extraction from a live KiteVM.
// t is the Go test handle used for assertions.
// The test protects sshId, Service, and Secret mapping used by password authentication and backend dialing.
func TestRouteFromKiteVirtualMachineUsesStatusNames(t *testing.T) {
	vm := newRouteTestKiteVM("user-a", "vm-a", "asdf", auth.HashPassword("password", routeTestPasswordSalt))
	if err := unstructured.SetNestedField(vm.Object, "custom-service", "status", "serviceName"); err != nil {
		t.Fatalf("failed to set status.serviceName: %v", err)
	}
	if err := unstructured.SetNestedField(vm.Object, "custom-secret", "status", "sshKeySecretName"); err != nil {
		t.Fatalf("failed to set status.sshKeySecretName: %v", err)
	}

	route, ok := RouteFromKiteVirtualMachine(vm)
	if !ok {
		t.Fatal("expected route to be extracted")
	}
	if route.Username != "asdf" || !auth.VerifyPassword("password", routeTestPasswordSalt, route.PasswordHash) {
		t.Fatalf("unexpected auth route: %#v", route)
	}
	if route.ServiceName != "custom-service" || route.SecretName != "custom-secret" {
		t.Fatalf("expected status names, got service=%q secret=%q", route.ServiceName, route.SecretName)
	}
	if route.TargetAddress() != "custom-service.user-a.svc.cluster.local:22" {
		t.Fatalf("unexpected target address %q", route.TargetAddress())
	}
}

// TestRouteFromKiteVirtualMachineSkipsDeleteIntent verifies deleting VMs cannot authenticate.
// t is the Go test handle used for assertions.
// The test keeps kite-gateway from routing users into VMs marked for deletion.
func TestRouteFromKiteVirtualMachineSkipsDeleteIntent(t *testing.T) {
	vm := newRouteTestKiteVM("user-a", "vm-a", "asdf", auth.HashPassword("password", routeTestPasswordSalt))
	if err := unstructured.SetNestedField(vm.Object, true, "spec", "delete"); err != nil {
		t.Fatalf("failed to set spec.delete: %v", err)
	}

	if _, ok := RouteFromKiteVirtualMachine(vm); ok {
		t.Fatal("expected delete-intended VM to be skipped")
	}
}

// TestRouteFromKiteVirtualMachineSkipsUnsafeSSHID verifies unsafe sshId values cannot become routes.
// t is the Go test handle used for assertions.
// The test keeps malformed usernames from reaching SSH authentication and backend dialing.
func TestRouteFromKiteVirtualMachineSkipsUnsafeSSHID(t *testing.T) {
	vm := newRouteTestKiteVM("user-a", "vm-a", "asdf;whoami", auth.HashPassword("password", routeTestPasswordSalt))

	if _, ok := RouteFromKiteVirtualMachine(vm); ok {
		t.Fatal("expected unsafe sshId to be skipped")
	}
}

// TestRouteFromKiteVirtualMachineSkipsUnsafeServiceName verifies unsafe status service names cannot become targets.
// t is the Go test handle used for assertions.
// The test keeps invalid backend DNS names out of the gateway route table.
func TestRouteFromKiteVirtualMachineSkipsUnsafeServiceName(t *testing.T) {
	vm := newRouteTestKiteVM("user-a", "vm-a", "asdf", auth.HashPassword("password", routeTestPasswordSalt))
	if err := unstructured.SetNestedField(vm.Object, "bad service", "status", "serviceName"); err != nil {
		t.Fatalf("failed to set status.serviceName: %v", err)
	}

	if _, ok := RouteFromKiteVirtualMachine(vm); ok {
		t.Fatal("expected unsafe serviceName to be skipped")
	}
}

// TestRouteTableRejectsDuplicateSSHID verifies duplicate sshId values are rejected.
// t is the Go test handle used for assertions.
// The test protects the global sshId uniqueness assumption used by v1 routing.
func TestRouteTableRejectsDuplicateSSHID(t *testing.T) {
	table := NewRouteTable(routeTestPasswordSalt)
	table.ReplaceAll([]Route{
		{Username: "asdf", PasswordHash: auth.HashPassword("one", routeTestPasswordSalt), VMNamespace: "user-a", VMName: "vm-a"},
		{Username: "asdf", PasswordHash: auth.HashPassword("two", routeTestPasswordSalt), VMNamespace: "user-b", VMName: "vm-b"},
	})

	if _, err := table.AuthenticatePassword("asdf", []byte("one")); err != ErrRouteDuplicate {
		t.Fatalf("expected ErrRouteDuplicate, got %v", err)
	}
}

func newRouteTestKiteVM(namespace string, name string, sshID string, passwordHash string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":              name,
				"namespace":         namespace,
				"creationTimestamp": metav1.Now().Format("2006-01-02T15:04:05Z07:00"),
			},
			"spec": map[string]any{
				"sshId":           sshID,
				"sshPasswordHash": passwordHash,
			},
		},
	}
}
