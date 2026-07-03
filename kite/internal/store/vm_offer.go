package store

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// KiteVirtualMachineOfferSpec contains the spec fields stored in the KiteVirtualMachineOffer CRD.
// These values are assigned by a level 3 admin and completed by a user during claim.
type KiteVirtualMachineOfferSpec struct {
	CPU       int
	Memory    string
	Image     string
	Disk      string
	ExpiresAt string
	CreatedBy string
}

// KiteVirtualMachineOfferRecord contains metadata and spec values for a namespaced offer CRD.
// Name is metadata.name and Namespace is the target user's namespace.
// Spec contains the admin-assigned VM capacity.
type KiteVirtualMachineOfferRecord struct {
	Name      string
	Namespace string
	Spec      KiteVirtualMachineOfferSpec
}

// VirtualMachineOfferStore provides CRUD access to KiteVirtualMachineOffer custom resources.
// dynamicClient is used because Kite CRDs are stored as unstructured Kubernetes resources.
// This store is expected to be used by kite-api offer handlers and controller cleanup code.
type VirtualMachineOfferStore struct {
	dynamicClient dynamic.Interface
}

// NewVirtualMachineOfferStore creates a KiteVirtualMachineOffer CRD store.
// dynamicClient is the Kubernetes dynamic client used to read and write offer resources.
// The returned store is used by API handlers before an offer claim creates a real VM CRD.
func NewVirtualMachineOfferStore(dynamicClient dynamic.Interface) *VirtualMachineOfferStore {
	return &VirtualMachineOfferStore{dynamicClient: dynamicClient}
}

// Create creates a namespaced KiteVirtualMachineOffer custom resource.
// ctx controls the Kubernetes API request lifetime.
// record provides metadata.name, metadata.namespace, and spec fields for the new offer.
// The returned object is the offer resource created by the Kubernetes API server.
func (s *VirtualMachineOfferStore) Create(ctx context.Context, record KiteVirtualMachineOfferRecord) (*unstructured.Unstructured, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace(record.Namespace).Create(ctx, newKiteVirtualMachineOfferObject(record), metav1.CreateOptions{})
}

// Get reads a namespaced KiteVirtualMachineOffer custom resource by namespace and name.
// ctx controls the Kubernetes API request lifetime.
// namespace is metadata.namespace of the offer resource and name is metadata.name.
// The returned object is the current offer stored in Kubernetes.
func (s *VirtualMachineOfferStore) Get(ctx context.Context, namespace string, name string) (*unstructured.Unstructured, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List reads KiteVirtualMachineOffer custom resources from one namespace.
// ctx controls the Kubernetes API request lifetime.
// namespace scopes the list request to one target user's namespace.
// The returned list contains visible unclaimed offers for that namespace.
func (s *VirtualMachineOfferStore) List(ctx context.Context, namespace string) (*unstructured.UnstructuredList, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

// ListAll reads KiteVirtualMachineOffer custom resources across every namespace.
// ctx controls the Kubernetes API request lifetime.
// The returned list is used by controller expiry cleanup.
func (s *VirtualMachineOfferStore) ListAll(ctx context.Context) (*unstructured.UnstructuredList, error) {
	return s.dynamicClient.Resource(kiteVirtualMachineOfferGVR).List(ctx, metav1.ListOptions{})
}

// UpdateStatus writes status fields for one offer using the current resource version.
// ctx controls the Kubernetes API request lifetime.
// current is the latest offer object and status is the next status map.
// The returned object is the offer resource after the status update.
func (s *VirtualMachineOfferStore) UpdateStatus(ctx context.Context, current *unstructured.Unstructured, status map[string]any) (*unstructured.Unstructured, error) {
	next := current.DeepCopy()
	next.Object["status"] = status
	return s.dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace(current.GetNamespace()).UpdateStatus(ctx, next, metav1.UpdateOptions{})
}

// ctx controls the Kubernetes API request lifetime.
// namespace and name identify the offer resource to remove.
// A nil error means Kubernetes accepted the delete request.
func (s *VirtualMachineOfferStore) Delete(ctx context.Context, namespace string, name string) error {
	return s.dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// newKiteVirtualMachineOfferObject converts an offer record into an unstructured CRD object.
// record provides Kubernetes metadata and desired offer spec fields.
// The returned object is passed directly to the Kubernetes dynamic client.
func newKiteVirtualMachineOfferObject(record KiteVirtualMachineOfferRecord) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachineOffer",
			"metadata": map[string]any{
				"name":      record.Name,
				"namespace": record.Namespace,
			},
			"spec": map[string]any{
				"cpu":       int64(record.Spec.CPU),
				"memory":    record.Spec.Memory,
				"image":     record.Spec.Image,
				"disk":      record.Spec.Disk,
				"expiresAt": record.Spec.ExpiresAt,
				"createdBy": record.Spec.CreatedBy,
			},
		},
	}
}
