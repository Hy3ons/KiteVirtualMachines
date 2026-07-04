package apps

import (
	"context"
	"encoding/base64"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/guestlogin"
	"kite/internal/platform"
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

// TestDeleteOwnedNamespacedResourceSkipsUnlabeledResource verifies destructive cleanup label guards.
// t is the Go test handle used for assertions.
// The test protects golden resources such as kite/ubuntu-22.04 from VM cleanup by name alone.
func TestDeleteOwnedNamespacedResourceSkipsUnlabeledResource(t *testing.T) {
	ctx := context.Background()
	namespace := "kite"
	name := "ubuntu-22.04"
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcileDataVolume(namespace, name, nil),
	)

	if err := deleteOwnedNamespacedResource(ctx, client, dataVolumeGVR, namespace, name, namespace, name); err != nil {
		t.Fatalf("deleteOwnedNamespacedResource returned error: %v", err)
	}

	if _, err := client.Resource(dataVolumeGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
		t.Fatalf("expected unlabeled DataVolume to remain, got %v", err)
	}
}

// TestDeleteOwnedNamespacedResourceDeletesMatchingOwner verifies labeled child cleanup.
// t is the Go test handle used for assertions.
// The test ensures the label guard still deletes resources explicitly owned by the KiteVM.
func TestDeleteOwnedNamespacedResourceDeletesMatchingOwner(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	name := "vm-a-disk"
	ownerName := "vm-a"
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcileDataVolume(namespace, name, kiteOwnerLabels(namespace, ownerName)),
	)

	if err := deleteOwnedNamespacedResource(ctx, client, dataVolumeGVR, namespace, name, namespace, ownerName); err != nil {
		t.Fatalf("deleteOwnedNamespacedResource returned error: %v", err)
	}

	if _, err := client.Resource(dataVolumeGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected labeled DataVolume to be deleted, got %v", err)
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
		newMachineReconcileGuestLoginSecret(namespace, name, "$6$rounds=500000$salt$hash"),
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

// TestKiteVirtualMachineNeedsSameGenerationReconcileRetriesFailedPhase verifies dependency retry.
// t is the Go test handle used for assertions.
// The test keeps failed VM reconciliation retryable when KubeVirt, CDI, or golden images become ready later.
func TestKiteVirtualMachineNeedsSameGenerationReconcileRetriesFailedPhase(t *testing.T) {
	vm := newMachineReconcileKiteVirtualMachineSpec("user-a", "vm-a")
	if err := unstructured.SetNestedField(vm.Object, kiteVMPhaseFailed, "status", "phase"); err != nil {
		t.Fatalf("failed to set status.phase: %v", err)
	}

	if !kiteVirtualMachineNeedsSameGenerationReconcile(vm) {
		t.Fatal("expected Failed phase to retry on same-generation resync")
	}
}

// TestKiteVirtualMachineNeedsSameGenerationReconcileSkipsReadyPhase verifies stable VM updates.
// t is the Go test handle used for assertions.
// The test prevents stable Ready VMs from being re-applied on every informer resync.
func TestKiteVirtualMachineNeedsSameGenerationReconcileSkipsReadyPhase(t *testing.T) {
	vm := newMachineReconcileKiteVirtualMachineSpec("user-a", "vm-a")
	if err := unstructured.SetNestedField(vm.Object, kiteVMPhaseReady, "status", "phase"); err != nil {
		t.Fatalf("failed to set status.phase: %v", err)
	}

	if kiteVirtualMachineNeedsSameGenerationReconcile(vm) {
		t.Fatal("expected Ready phase to skip same-generation resync")
	}
}

// TestUpdateKiteVirtualMachineStatusPreservesStableRuntimePhase verifies same-generation dependency updates.
// t is the Go test handle used for assertions.
// The test prevents desired, Service, or DataVolume reconcilers from moving a stopped VM back to Provisioning.
func TestUpdateKiteVirtualMachineStatusPreservesStableRuntimePhase(t *testing.T) {
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
	if err := updateKiteVirtualMachineStatus(ctx, client, typedVM, kiteVMPhaseProvisioning, "Off", "", "", metav1.ConditionFalse, kiteVMReasonReconciled, "VM resources are applied and waiting for KubeVirt readiness"); err != nil {
		t.Fatalf("updateKiteVirtualMachineStatus returned error: %v", err)
	}

	current, err := client.Resource(kiteVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read updated VM: %v", err)
	}
	phase, _, _ := unstructured.NestedString(current.Object, "status", "phase")
	message := firstStatusConditionMessage(current)
	if phase != kiteVMPhaseStopped {
		t.Fatalf("expected stable phase %q to be preserved, got %q", kiteVMPhaseStopped, phase)
	}
	if message != "KubeVirt VirtualMachine is stopped" {
		t.Fatalf("expected stable condition message to be preserved, got %q", message)
	}
}

// TestUpdateKiteVirtualMachineStatusAllowsProvisioningForNewGeneration verifies real spec changes can transition.
// t is the Go test handle used for assertions.
// The test keeps power-state changes from being hidden by the stable-phase preservation guard.
func TestUpdateKiteVirtualMachineStatusAllowsProvisioningForNewGeneration(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	name := "vm-a"
	vm := newMachineReconcileKiteVirtualMachineSpec(namespace, name)
	vm.Object["metadata"].(map[string]any)["generation"] = int64(2)
	setMachineReconcileStatus(t, vm, kiteVMPhaseStopped, "Off", 1, metav1.ConditionTrue, kiteVMReasonReconciled, "KubeVirt VirtualMachine is stopped")
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(), vm)

	typedVM, err := kiteVirtualMachineFromEventObject(vm)
	if err != nil {
		t.Fatalf("failed to convert test VM: %v", err)
	}
	if err := updateKiteVirtualMachineStatus(ctx, client, typedVM, kiteVMPhaseProvisioning, "On", "", "", metav1.ConditionFalse, kiteVMReasonReconciled, "KubeVirt VirtualMachine is starting"); err != nil {
		t.Fatalf("updateKiteVirtualMachineStatus returned error: %v", err)
	}

	current, err := client.Resource(kiteVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to read updated VM: %v", err)
	}
	phase, _, _ := unstructured.NestedString(current.Object, "status", "phase")
	if phase != kiteVMPhaseProvisioning {
		t.Fatalf("expected new generation to move to %q, got %q", kiteVMPhaseProvisioning, phase)
	}
}

// TestKiteVirtualMachineStorageClassNameDefaultsWhenConfigMissing verifies the Longhorn default.
// t is the Go test handle used for assertions.
// The test is used by VM reconcile code when kite-runtime-config has not been created yet.
func TestKiteVirtualMachineStorageClassNameDefaultsWhenConfigMissing(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds())

	storageClassName, err := kiteVirtualMachineStorageClassName(ctx, client)
	if err != nil {
		t.Fatalf("kiteVirtualMachineStorageClassName returned error: %v", err)
	}
	if storageClassName != kiteDefaultVMStorageClassName {
		t.Fatalf("expected default storage class %q, got %q", kiteDefaultVMStorageClassName, storageClassName)
	}
}

// TestKiteVirtualMachineStorageClassNameReadsRuntimeConfig verifies ConfigMap override handling.
// t is the Go test handle used for assertions.
// The test is used by VM reconcile code that renders DataVolumes from production runtime config.
func TestKiteVirtualMachineStorageClassNameReadsRuntimeConfig(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcileRuntimeConfig("custom-storage"),
	)

	storageClassName, err := kiteVirtualMachineStorageClassName(ctx, client)
	if err != nil {
		t.Fatalf("kiteVirtualMachineStorageClassName returned error: %v", err)
	}
	if storageClassName != "custom-storage" {
		t.Fatalf("expected configured storage class, got %q", storageClassName)
	}
}

// TestEnsureKiteVirtualMachineIngressTLSSecretCreatesNamespaceCopy verifies VM Ingress TLS wiring.
// t is the Go test handle used for assertions.
// The test ensures global TLS material is copied into the user namespace because Ingress TLS Secrets are namespace-local.
func TestEnsureKiteVirtualMachineIngressTLSSecretCreatesNamespaceCopy(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcilePlatformTLSSecret(),
	)

	secretName, err := ensureKiteVirtualMachineIngressTLSSecret(ctx, client, namespace)
	if err != nil {
		t.Fatalf("ensureKiteVirtualMachineIngressTLSSecret returned error: %v", err)
	}

	if secretName != platform.GlobalTLSSecretName {
		t.Fatalf("expected TLS secret name %q, got %q", platform.GlobalTLSSecretName, secretName)
	}
	copied, err := client.Resource(secretGVR).Namespace(namespace).Get(ctx, platform.GlobalTLSSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected namespace TLS Secret to be created, got %v", err)
	}
	labels := copied.GetLabels()
	if labels[kiteSecretTypeLabel] != kitePlatformTLSSecretType {
		t.Fatalf("expected Kite-managed TLS Secret label, got %#v", labels)
	}
}

// TestEnsureKiteVirtualMachineIngressTLSSecretSkipsWhenGlobalSecretMissing verifies optional TLS behavior.
// t is the Go test handle used for assertions.
// The test keeps VM domain Ingress usable over HTTP before an admin uploads a TLS certificate.
func TestEnsureKiteVirtualMachineIngressTLSSecretSkipsWhenGlobalSecretMissing(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds())

	secretName, err := ensureKiteVirtualMachineIngressTLSSecret(ctx, client, "user-a")
	if err != nil {
		t.Fatalf("ensureKiteVirtualMachineIngressTLSSecret returned error: %v", err)
	}
	if secretName != "" {
		t.Fatalf("expected no TLS secret name without global TLS material, got %q", secretName)
	}
}

// TestEnsureKiteVirtualMachineIngressTLSSecretRejectsUnmanagedNamespaceSecret verifies overwrite protection.
// t is the Go test handle used for assertions.
// The test prevents Kite from replacing an operator-owned Secret with the global TLS certificate.
func TestEnsureKiteVirtualMachineIngressTLSSecretRejectsUnmanagedNamespaceSecret(t *testing.T) {
	ctx := context.Background()
	namespace := "user-a"
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), machineReconcileListKinds(),
		newMachineReconcilePlatformTLSSecret(),
		newMachineReconcileUnmanagedTLSSecret(namespace),
	)

	secretName, err := ensureKiteVirtualMachineIngressTLSSecret(ctx, client, namespace)
	if err == nil {
		t.Fatalf("expected unmanaged TLS Secret to fail, got secretName=%q", secretName)
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

// setMachineReconcileStatus writes VM status fields used by reconcile tests.
// t is the Go test handle used for assertion failures.
// obj is the unstructured KiteVirtualMachine under test.
// phase, powerState, observedGeneration, conditionStatus, reason, and message model controller status.
func setMachineReconcileStatus(t *testing.T, obj *unstructured.Unstructured, phase string, powerState string, observedGeneration int64, conditionStatus metav1.ConditionStatus, reason string, message string) {
	t.Helper()
	if err := unstructured.SetNestedField(obj.Object, phase, "status", "phase"); err != nil {
		t.Fatalf("failed to set status.phase: %v", err)
	}
	if err := unstructured.SetNestedField(obj.Object, powerState, "status", "currentPowerState"); err != nil {
		t.Fatalf("failed to set status.currentPowerState: %v", err)
	}
	if err := unstructured.SetNestedField(obj.Object, observedGeneration, "status", "observedGeneration"); err != nil {
		t.Fatalf("failed to set status.observedGeneration: %v", err)
	}
	conditions := []any{kiteVirtualMachineCondition(observedGeneration, conditionStatus, reason, message)}
	if err := unstructured.SetNestedSlice(obj.Object, conditions, "status", "conditions"); err != nil {
		t.Fatalf("failed to set status.conditions: %v", err)
	}
}

// machineReconcileListKinds returns fake dynamic client list kind mappings.
// The returned map lets the fake client list Kite and KubeVirt VM resources if a test needs it.
// This helper is used by machine reconcile unit tests.
func machineReconcileListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		kiteVirtualMachineGVR:     "KiteVirtualMachineList",
		kubeVirtVirtualMachineGVR: "VirtualMachineList",
		dataVolumeGVR:             "DataVolumeList",
		secretGVR:                 "SecretList",
		configMapGVR:              "ConfigMapList",
	}
}

// kiteOwnerLabels returns labels used by controller-rendered VM child resources.
// namespace is the KiteVirtualMachine namespace.
// name is the KiteVirtualMachine metadata.name.
// The returned map is used by tests that exercise deletion ownership guards.
func kiteOwnerLabels(namespace string, name string) map[string]any {
	return map[string]any{
		kiteManagedByLabel:   kiteControllerLabel,
		kiteVMNamespaceLabel: namespace,
		kiteVMNameLabel:      name,
	}
}

// newMachineReconcileKiteVirtualMachine creates an unstructured KiteVirtualMachine test object.
// namespace is metadata.namespace for the namespaced CRD.
// name is metadata.name for both Kite and KubeVirt VM resources.
// deleteIntent controls spec.delete in the returned object.
func newMachineReconcileKiteVirtualMachine(namespace string, name string, deleteIntent bool) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
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
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":       name,
				"namespace":  namespace,
				"generation": int64(1),
			},
			"spec": map[string]any{
				"cpu":             int64(2),
				"memory":          "4Gi",
				"image":           "ubuntu-22.04",
				"disk":            "25Gi",
				"powerState":      "Off",
				"sshId":           "ubuntu",
				"sshPasswordHash": "hashed-password",
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
				"labels":    kiteOwnerLabels(namespace, name),
			},
		},
	}
}

// newMachineReconcileDataVolume creates an unstructured CDI DataVolume test object.
// namespace is metadata.namespace for the DataVolume.
// name is metadata.name for the DataVolume.
// labels are optional metadata.labels used to model managed and unmanaged resources.
func newMachineReconcileDataVolume(namespace string, name string, labels map[string]any) *unstructured.Unstructured {
	metadata := map[string]any{
		"name":      name,
		"namespace": namespace,
	}
	if labels != nil {
		metadata["labels"] = labels
	}

	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "cdi.kubevirt.io/v1beta1",
			"kind":       "DataVolume",
			"metadata":   metadata,
		},
	}
}

// newMachineReconcileGuestLoginSecret creates a guest login Secret test fixture.
// namespace is metadata.namespace for the Secret.
// name is the owning KiteVirtualMachine name used to derive the Secret name and labels.
// passwordHash is the Linux crypt hash that controller code reads for cloud-init.
// The returned object is used by VM reconcile tests that need to pass guest login setup.
func newMachineReconcileGuestLoginSecret(namespace string, name string, passwordHash string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      guestlogin.SecretName(name),
				"namespace": namespace,
				"labels":    kiteOwnerLabels(namespace, name),
			},
			"data": map[string]any{
				guestlogin.PasswordHashKey: base64.StdEncoding.EncodeToString([]byte(passwordHash)),
			},
		},
	}
}

// newMachineReconcilePlatformTLSSecret creates the global TLS Secret test fixture.
// The returned Secret uses the same namespace and name that platform settings writes in production.
// Tests use it to verify VM namespace TLS copy behavior.
func newMachineReconcilePlatformTLSSecret() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      platform.GlobalTLSSecretName,
				"namespace": platform.GlobalTLSSecretNS,
			},
			"type": "kubernetes.io/tls",
			"data": map[string]any{
				platform.TLSCertificateKey:    "Y2VydA==",
				platform.TLSPrivateKeyDataKey: "a2V5",
			},
		},
	}
}

// newMachineReconcileUnmanagedTLSSecret creates an operator-owned TLS Secret test fixture.
// namespace is the user namespace where a VM Ingress will be rendered.
// The returned object intentionally lacks Kite labels so overwrite protection can be tested.
func newMachineReconcileUnmanagedTLSSecret(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      platform.GlobalTLSSecretName,
				"namespace": namespace,
			},
			"type": "kubernetes.io/tls",
			"data": map[string]any{
				platform.TLSCertificateKey:    "b3BlcmF0b3ItY2VydA==",
				platform.TLSPrivateKeyDataKey: "b3BlcmF0b3Ita2V5",
			},
		},
	}
}

// newMachineReconcileRuntimeConfig creates a kite-runtime-config test ConfigMap.
// storageClassName is stored in data.vmStorageClassName.
// The returned object is used by storage configuration tests.
func newMachineReconcileRuntimeConfig(storageClassName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      kiteGlobalConfigName,
				"namespace": kiteGlobalConfigNamespace,
			},
			"data": map[string]any{
				kiteGlobalVMStorageClassNameKey: storageClassName,
			},
		},
	}
}
