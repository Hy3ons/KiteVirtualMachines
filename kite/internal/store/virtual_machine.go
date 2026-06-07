package store

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// KiteVirtualMachineSpec contains the spec fields stored in the KiteVirtualMachine CRD.
// These values come from API VM requests and describe the desired VM state.
type KiteVirtualMachineSpec struct {
	CPU          int
	Memory       string
	Image        string
	Disk         string
	PowerState   string
	DomainPrefix string
	SSHID        string
	SSHPassword  string
	Delete       bool
}

// KiteVirtualMachineRecord contains metadata and spec values for a namespaced KiteVirtualMachine CRD.
// Name is used as metadata.name.
// Namespace is used as metadata.namespace and the dynamic client namespace.
// Spec contains the desired VM configuration.
type KiteVirtualMachineRecord struct {
	Name      string
	Namespace string
	Spec      KiteVirtualMachineSpec
}

// VirtualMachineStore provides CRUD access to KiteVirtualMachine custom resources.
// dynamicClient is used because Kite CRDs are stored as unstructured Kubernetes resources.
// This store is expected to be used by kite-api handlers before the controller reconciles real VM resources.
type VirtualMachineStore struct {
	dynamicClient dynamic.Interface
}

// NewVirtualMachineStore creates a KiteVirtualMachine CRD store.
// dynamicClient is the Kubernetes dynamic client used to read and write KiteVirtualMachine resources.
// The returned store is used by API handlers that treat KiteVirtualMachine CRDs as the backing store.
func NewVirtualMachineStore(dynamicClient dynamic.Interface) *VirtualMachineStore {
	return &VirtualMachineStore{
		dynamicClient: dynamicClient,
	}
}

// Create creates a namespaced KiteVirtualMachine custom resource.
// ctx controls the Kubernetes API request lifetime.
// record provides metadata.name, metadata.namespace, and spec fields for the new resource.
// The returned object is the KiteVirtualMachine resource created by the Kubernetes API server.
func (s *VirtualMachineStore) Create(ctx context.Context, record KiteVirtualMachineRecord) (*unstructured.Unstructured, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(record.Namespace).Create(ctx, newKiteVirtualMachineObject(record), metav1.CreateOptions{})
}

// Get reads a namespaced KiteVirtualMachine custom resource by namespace and name.
// ctx controls the Kubernetes API request lifetime.
// namespace is metadata.namespace of the VM resource.
// name is metadata.name of the VM resource.
// The returned object is the current KiteVirtualMachine stored in Kubernetes.
func (s *VirtualMachineStore) Get(ctx context.Context, namespace string, name string) (*unstructured.Unstructured, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List reads KiteVirtualMachine custom resources from one namespace.
// ctx controls the Kubernetes API request lifetime.
// namespace scopes the list request to one Kubernetes namespace.
// The returned list contains the KiteVirtualMachine resources in that namespace.
func (s *VirtualMachineStore) List(ctx context.Context, namespace string) (*unstructured.UnstructuredList, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

// ListAll reads KiteVirtualMachine custom resources across every namespace.
// ctx controls the Kubernetes API request lifetime.
// The returned list is used by admin API handlers that need cluster-wide VM visibility.
func (s *VirtualMachineStore) ListAll(ctx context.Context) (*unstructured.UnstructuredList, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineGVR).List(ctx, metav1.ListOptions{})
}

// Update replaces the spec of an existing namespaced KiteVirtualMachine custom resource.
// ctx controls the Kubernetes API request lifetime.
// record provides metadata.name, metadata.namespace, and desired spec values.
// The returned object is the KiteVirtualMachine resource after the Kubernetes API update.
func (s *VirtualMachineStore) Update(ctx context.Context, record KiteVirtualMachineRecord) (*unstructured.Unstructured, error) {
	current, err := s.Get(ctx, record.Namespace, record.Name)
	if err != nil {
		return nil, err
	}

	next := current.DeepCopy()
	next.Object["spec"] = kiteVirtualMachineSpecMap(record.Spec)

	return s.dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(record.Namespace).Update(ctx, next, metav1.UpdateOptions{})
}

// Delete deletes a namespaced KiteVirtualMachine custom resource.
// ctx controls the Kubernetes API request lifetime.
// namespace is metadata.namespace of the VM resource.
// name is metadata.name of the VM resource.
// A nil error means Kubernetes accepted the delete request.
func (s *VirtualMachineStore) Delete(ctx context.Context, namespace string, name string) error {
	return s.dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// newKiteVirtualMachineObject converts a KiteVirtualMachineRecord into an unstructured CRD object.
// record provides Kubernetes metadata and desired VM spec fields.
// The returned object is passed directly to the Kubernetes dynamic client by VirtualMachineStore.
func newKiteVirtualMachineObject(record KiteVirtualMachineRecord) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":      record.Name,
				"namespace": record.Namespace,
			},
			"spec": kiteVirtualMachineSpecMap(record.Spec),
		},
	}
}

// kiteVirtualMachineSpecMap converts a store VM spec into an unstructured CRD spec map.
// spec contains user desired state values accepted by the API.
// The returned map is used for create and spec-only updates.
func kiteVirtualMachineSpecMap(spec KiteVirtualMachineSpec) map[string]any {
	return map[string]any{
		"cpu":          int64(spec.CPU),
		"memory":       spec.Memory,
		"image":        spec.Image,
		"disk":         spec.Disk,
		"powerState":   spec.PowerState,
		"domainPrefix": spec.DomainPrefix,
		"sshId":        spec.SSHID,
		"sshPassword":  spec.SSHPassword,
		"delete":       spec.Delete,
	}
}
