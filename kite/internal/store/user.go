package store

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// KiteUserSpec contains the spec fields stored in the KiteUser CRD.
// These values come from API user requests and are written to custom/kite-user-crd.yaml resources.
type KiteUserSpec struct {
	Username     string
	Email        string
	Password     string
	Namespace    string
	ProfileImage string
	AccessLevel  int
}

// KiteUserRecord contains the Kubernetes metadata and spec needed to store a KiteUser CRD.
// Name is used as metadata.name for the cluster-scoped KiteUser resource.
// Spec contains the user-facing fields that become spec values in Kubernetes.
type KiteUserRecord struct {
	Name string
	Spec KiteUserSpec
}

// UserStore provides CRUD access to KiteUser custom resources.
// dynamicClient is used because Kite CRDs are stored as unstructured Kubernetes resources.
// This store is expected to be used by kite-api handlers, not by controller reconcile code.
type UserStore struct {
	dynamicClient dynamic.Interface
}

// NewUserStore creates a KiteUser CRD store.
// dynamicClient is the Kubernetes dynamic client used to read and write KiteUser resources.
// The returned store is used by API handlers that treat KiteUser CRDs as the backing store.
func NewUserStore(dynamicClient dynamic.Interface) *UserStore {
	return &UserStore{
		dynamicClient: dynamicClient,
	}
}

// Create creates a cluster-scoped KiteUser custom resource.
// ctx controls the Kubernetes API request lifetime.
// record provides metadata.name and spec fields for the new KiteUser.
// The returned object is the KiteUser resource created by the Kubernetes API server.
func (s *UserStore) Create(ctx context.Context, record KiteUserRecord) (*unstructured.Unstructured, error) {
	return s.dynamicClient.Resource(kiteUserGVR).Create(ctx, newKiteUserObject(record), metav1.CreateOptions{})
}

// Get reads a cluster-scoped KiteUser custom resource by name.
// ctx controls the Kubernetes API request lifetime.
// name is metadata.name of the KiteUser resource.
// The returned object is the current KiteUser stored in Kubernetes.
func (s *UserStore) Get(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	return s.dynamicClient.Resource(kiteUserGVR).Get(ctx, name, metav1.GetOptions{})
}

// List reads all cluster-scoped KiteUser custom resources.
// ctx controls the Kubernetes API request lifetime.
// The returned list contains every KiteUser visible to the dynamic client.
func (s *UserStore) List(ctx context.Context) (*unstructured.UnstructuredList, error) {
	return s.dynamicClient.Resource(kiteUserGVR).List(ctx, metav1.ListOptions{})
}

// Update replaces the spec of an existing cluster-scoped KiteUser custom resource.
// ctx controls the Kubernetes API request lifetime.
// record provides metadata.name and the desired spec values.
// The returned object is the KiteUser resource after the Kubernetes API update.
func (s *UserStore) Update(ctx context.Context, record KiteUserRecord) (*unstructured.Unstructured, error) {
	current, err := s.Get(ctx, record.Name)
	if err != nil {
		return nil, err
	}

	next := newKiteUserObject(record)
	next.SetResourceVersion(current.GetResourceVersion())

	return s.dynamicClient.Resource(kiteUserGVR).Update(ctx, next, metav1.UpdateOptions{})
}

// Delete deletes a cluster-scoped KiteUser custom resource by name.
// ctx controls the Kubernetes API request lifetime.
// name is metadata.name of the KiteUser resource to delete.
// A nil error means Kubernetes accepted the delete request.
func (s *UserStore) Delete(ctx context.Context, name string) error {
	return s.dynamicClient.Resource(kiteUserGVR).Delete(ctx, name, metav1.DeleteOptions{})
}

// newKiteUserObject converts a KiteUserRecord into an unstructured KiteUser CRD object.
// record provides Kubernetes metadata and CRD spec fields.
// The returned object is passed directly to the Kubernetes dynamic client by UserStore.
func newKiteUserObject(record KiteUserRecord) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "anacnu.com/v1",
			"kind":       "KiteUser",
			"metadata": map[string]any{
				"name": record.Name,
			},
			"spec": map[string]any{
				"username":      record.Spec.Username,
				"email":         record.Spec.Email,
				"password":      record.Spec.Password,
				"namespace":     record.Spec.Namespace,
				"profile_image": record.Spec.ProfileImage,
				"access_level":  int64(record.Spec.AccessLevel),
			},
		},
	}
}
