package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"

	kitev1 "kite/api/v1"
	"kite/internal/kube"
	"kite/internal/render/namespace"
	networkpolicy "kite/internal/render/network-policy"
	quotapolicy "kite/internal/render/quota-policy"
)

const userQuotaPolicyName = "kite-user-quota-policy"

const userReconcileResyncPeriod = time.Minute * 3

const (
	kiteUserPhaseReady         = "Ready"
	kiteUserPhaseFailed        = "Failed"
	kiteUserConditionReady     = "NamespaceReady"
	kiteUserReasonReconciled   = "Reconciled"
	kiteUserReasonFailed       = "ReconcileFailed"
	kiteUserReadyMessage       = "user namespace, network policies, and resource quota are ready"
	kiteUserStatusFieldManager = "kite-controller-user-status"
	kiteNamespaceManagedByKey  = "hy3ons.github.io/managed-by"
	kiteNamespaceManagedBy     = "kite-controller"
)

var (
	kiteUserGVR = schema.GroupVersionResource{
		Group:    "hy3ons.github.io",
		Version:  "v1",
		Resource: "kiteusers",
	}
	namespaceGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}
	resourceQuotaGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "resourcequotas",
	}
	networkPolicyGVR = schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "networkpolicies",
	}
)

// RunKiteUserReconciler creates and runs the KiteUser informer loop.
// clientManager provides the dynamic Kubernetes client used to watch KiteUser and apply user resources.
// stopCh stops the informer when the controller process is shutting down.
// This function is expected to run in a goroutine from cmd/kite-controller/main.go.
func RunKiteUserReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KiteUser reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, userReconcileResyncPeriod)
	informer := factory.ForResource(kiteUserGVR).Informer()
	RegisterKiteUserReconciler(informer, clientManager)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync KiteUser informer cache")
		return
	}

	<-stopCh
}

// RegisterKiteUserReconciler attaches KiteUser event handlers to an informer.
// informer watches cluster-scoped KiteUser custom resources from the controller startup code.
// clientManager provides the dynamic Kubernetes client used to apply namespace and policy resources.
// This function is used by cmd/kite-controller/main.go when wiring controller informers.
func RegisterKiteUserReconciler(informer cache.SharedIndexInformer, clientManager *kube.ClientManager) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ctx := context.Background()
			if err := ReconcileKiteUser(ctx, clientManager.DynamicClient, obj); err != nil {
				if statusErr := recordKiteUserReconcileFailure(ctx, clientManager.DynamicClient, obj, err); statusErr != nil {
					log.Printf("failed to reconcile KiteUser add event: %v; failed to update status: %v", err, statusErr)
					return
				}

				log.Printf("failed to reconcile KiteUser add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			ctx := context.Background()
			if err := ReconcileKiteUser(ctx, clientManager.DynamicClient, newObj); err != nil {
				if statusErr := recordKiteUserReconcileFailure(ctx, clientManager.DynamicClient, newObj, err); statusErr != nil {
					log.Printf("failed to reconcile KiteUser update event: %v; failed to update status: %v", err, statusErr)
					return
				}

				log.Printf("failed to reconcile KiteUser update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := DeleteKiteUserResources(context.Background(), clientManager.DynamicClient, obj); err != nil {
				log.Printf("failed to delete KiteUser resources: %v", err)
			}
		},
	})
}

// ReconcileKiteUser creates or updates the base Kubernetes resources for one KiteUser.
// ctx controls the Kubernetes API calls made during the reconcile operation.
// dynamicClient applies Namespace, NetworkPolicy, and ResourceQuota objects.
// eventObj is the informer event object for a KiteUser resource.
// This function is used by KiteUser add and update event handlers.
func ReconcileKiteUser(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	user, err := kiteUserFromEventObject(eventObj)
	if err != nil {
		return err
	}

	if user.Spec.Namespace == "" {
		return fmt.Errorf("KiteUser %s has empty spec.namespace", user.GetName())
	}

	if err := reconcileKiteUserNamespaceChange(ctx, dynamicClient, user); err != nil {
		return err
	}

	if err := validateUserNamespace(ctx, dynamicClient, user); err != nil {
		return err
	}

	objects, err := userBaseResources(user)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		if err := applyUserResource(ctx, dynamicClient, obj); err != nil {
			return err
		}
	}

	if err := updateKiteUserStatus(ctx, dynamicClient, user, kiteUserPhaseReady, metav1.ConditionTrue, kiteUserReasonReconciled, kiteUserReadyMessage); err != nil {
		return err
	}

	log.Printf("reconciled KiteUser %s namespace resources in %s", user.GetName(), user.Spec.Namespace)
	return nil
}

// recordKiteUserReconcileFailure writes a Failed condition to the KiteUser status.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient updates the KiteUser status subresource.
// eventObj is the informer event object that failed to reconcile.
// cause is the reconcile error that should be visible from kubectl describe.
func recordKiteUserReconcileFailure(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}, cause error) error {
	user, err := kiteUserFromEventObject(eventObj)
	if err != nil {
		return err
	}

	return updateKiteUserStatus(ctx, dynamicClient, user, kiteUserPhaseFailed, metav1.ConditionFalse, kiteUserReasonFailed, cause.Error())
}

// updateKiteUserStatus writes the latest KiteUser reconcile status.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient updates the cluster-scoped KiteUser status subresource.
// user provides the KiteUser name and observed generation.
// phase, conditionStatus, reason, and message are shown by kubectl get and kubectl describe.
func updateKiteUserStatus(ctx context.Context, dynamicClient dynamic.Interface, user *kitev1.KiteUser, phase string, conditionStatus metav1.ConditionStatus, reason string, message string) error {
	current, err := dynamicClient.Resource(kiteUserGVR).Get(ctx, user.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read KiteUser %s for status update: %w", user.GetName(), err)
	}

	if kiteUserStatusMatches(current, user.GetGeneration(), user.Spec.Namespace, phase, conditionStatus, reason, message) {
		return nil
	}

	condition := kiteUserStatusCondition(current, user.GetGeneration(), conditionStatus, reason, message)
	conditions := replaceKiteUserCondition(current, condition)

	if err := unstructured.SetNestedField(current.Object, phase, "status", "phase"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(current.Object, user.GetGeneration(), "status", "observedGeneration"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(current.Object, user.Spec.Namespace, "status", "observedNamespace"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(current.Object, message, "status", "message"); err != nil {
		return err
	}
	if err := unstructured.SetNestedSlice(current.Object, conditions, "status", "conditions"); err != nil {
		return err
	}

	if _, err := dynamicClient.Resource(kiteUserGVR).UpdateStatus(ctx, current, metav1.UpdateOptions{
		FieldManager: kiteUserStatusFieldManager,
	}); err != nil {
		return fmt.Errorf("failed to update KiteUser %s status: %w", user.GetName(), err)
	}

	return nil
}

// kiteUserStatusMatches checks whether the stored KiteUser status already has the desired value.
// current is the latest KiteUser object read from the Kubernetes API.
// observedGeneration is the metadata generation processed by this reconcile.
// namespace is the spec.namespace value that should be recorded as observedNamespace.
// phase, conditionStatus, reason, and message are the desired status values.
// The returned value prevents status-only update loops from informer events.
func kiteUserStatusMatches(current *unstructured.Unstructured, observedGeneration int64, namespace string, phase string, conditionStatus metav1.ConditionStatus, reason string, message string) bool {
	currentPhase, _, _ := unstructured.NestedString(current.Object, "status", "phase")
	currentObservedGeneration, _, _ := unstructured.NestedInt64(current.Object, "status", "observedGeneration")
	currentObservedNamespace, _, _ := unstructured.NestedString(current.Object, "status", "observedNamespace")
	currentMessage, _, _ := unstructured.NestedString(current.Object, "status", "message")
	condition := findKiteUserCondition(current)

	return currentPhase == phase &&
		currentObservedGeneration == observedGeneration &&
		currentObservedNamespace == namespace &&
		currentMessage == message &&
		stringValue(condition, "status") == string(conditionStatus) &&
		stringValue(condition, "reason") == reason &&
		stringValue(condition, "message") == message
}

// reconcileKiteUserNamespaceChange deletes the previously reconciled namespace after spec.namespace changes.
// ctx controls Kubernetes API calls.
// dynamicClient reads KiteUser status and deletes the old Namespace resource.
// user provides the desired spec.namespace and KiteUser name.
// A nil error means no previous namespace exists or the stale namespace cleanup was requested.
func reconcileKiteUserNamespaceChange(ctx context.Context, dynamicClient dynamic.Interface, user *kitev1.KiteUser) error {
	current, err := dynamicClient.Resource(kiteUserGVR).Get(ctx, user.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read KiteUser %s before namespace change cleanup: %w", user.GetName(), err)
	}

	previousNamespace, _, _ := unstructured.NestedString(current.Object, "status", "observedNamespace")
	if previousNamespace == "" || previousNamespace == user.Spec.Namespace {
		return nil
	}

	return deleteKiteUserNamespaceIfUnreferenced(ctx, dynamicClient, user.GetName(), previousNamespace)
}

// deleteKiteUserNamespaceIfUnreferenced deletes a stale Kite-managed namespace when no KiteUser references it.
// ctx controls Kubernetes API calls.
// dynamicClient reads KiteUsers and deletes core/v1 Namespace resources.
// userName is used only for controller logs.
// namespace is the previously reconciled namespace to remove.
func deleteKiteUserNamespaceIfUnreferenced(ctx context.Context, dynamicClient dynamic.Interface, userName string, namespace string) error {
	referenced, err := namespaceReferencedByKiteUser(ctx, dynamicClient, namespace)
	if err != nil {
		return err
	}
	if referenced {
		log.Printf("kept previous KiteUser %s namespace %s because another KiteUser still references it", userName, namespace)
		return nil
	}

	current, err := dynamicClient.Resource(namespaceGVR).Get(ctx, namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read previous KiteUser namespace %s: %w", namespace, err)
	}
	if !kiteManagedNamespace(current) {
		return fmt.Errorf("previous namespace %s is not managed by Kite", namespace)
	}

	err = dynamicClient.Resource(namespaceGVR).Delete(ctx, namespace, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete previous KiteUser namespace %s: %w", namespace, err)
	}

	log.Printf("deleted previous KiteUser %s namespace %s after spec.namespace changed", userName, namespace)
	return nil
}

// kiteUserStatusCondition builds the NamespaceReady condition stored in KiteUser status.
// current provides the existing condition so lastTransitionTime is stable when status does not change.
// observedGeneration is the KiteUser generation processed by the controller.
// conditionStatus, reason, and message describe the latest reconcile result.
func kiteUserStatusCondition(current *unstructured.Unstructured, observedGeneration int64, conditionStatus metav1.ConditionStatus, reason string, message string) map[string]any {
	existing := findKiteUserCondition(current)
	lastTransitionTime := metav1.Now().Format(time.RFC3339)
	if stringValue(existing, "status") == string(conditionStatus) {
		if existingTime := stringValue(existing, "lastTransitionTime"); existingTime != "" {
			lastTransitionTime = existingTime
		}
	}

	return map[string]any{
		"type":               kiteUserConditionReady,
		"status":             string(conditionStatus),
		"reason":             reason,
		"message":            message,
		"observedGeneration": observedGeneration,
		"lastTransitionTime": lastTransitionTime,
	}
}

// replaceKiteUserCondition replaces the NamespaceReady condition while preserving other conditions.
// current is the latest KiteUser object read from the Kubernetes API.
// condition is the new NamespaceReady condition map to store.
// The returned slice is written to status.conditions.
func replaceKiteUserCondition(current *unstructured.Unstructured, condition map[string]any) []any {
	existingConditions, _, _ := unstructured.NestedSlice(current.Object, "status", "conditions")
	conditions := make([]any, 0, len(existingConditions)+1)
	for _, item := range existingConditions {
		existing, ok := item.(map[string]any)
		if !ok || stringValue(existing, "type") == kiteUserConditionReady {
			continue
		}

		conditions = append(conditions, existing)
	}

	conditions = append(conditions, condition)
	return conditions
}

// findKiteUserCondition returns the stored NamespaceReady condition from a KiteUser object.
// current is the latest KiteUser object read from the Kubernetes API.
// The returned map is empty when no NamespaceReady condition exists.
func findKiteUserCondition(current *unstructured.Unstructured) map[string]any {
	conditions, _, _ := unstructured.NestedSlice(current.Object, "status", "conditions")
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if ok && stringValue(condition, "type") == kiteUserConditionReady {
			return condition
		}
	}

	return map[string]any{}
}

// stringValue reads a string field from unstructured status data.
// data is a map from a Kubernetes object field such as status.conditions.
// key is the field name to read.
// The returned value is empty when the field is missing or not a string.
func stringValue(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

// validateUserNamespace checks whether a KiteUser can manage its requested namespace.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient reads the cluster-scoped Namespace object before reconciliation applies resources.
// user provides spec.namespace, which is the namespace that KiteUser wants to manage.
// This helper fails reconciliation when the namespace already exists but was not created by Kite.
func validateUserNamespace(ctx context.Context, dynamicClient dynamic.Interface, user *kitev1.KiteUser) error {
	current, err := dynamicClient.Resource(namespaceGVR).Get(ctx, user.Spec.Namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read namespace %s for KiteUser %s: %w", user.Spec.Namespace, user.GetName(), err)
	}

	if kiteManagedNamespace(current) {
		return nil
	}

	return fmt.Errorf("namespace %s already exists and is not managed by Kite", user.Spec.Namespace)
}

// kiteManagedNamespace checks whether a namespace belongs to the Kite controller.
// namespace is the unstructured Namespace object read from the Kubernetes API.
// The returned value is true only when the Kite managed-by label is present.
// This helper prevents KiteUser reconcile and cleanup from taking over unrelated namespaces.
func kiteManagedNamespace(namespace *unstructured.Unstructured) bool {
	value, _, _ := unstructured.NestedString(namespace.Object, "metadata", "labels", kiteNamespaceManagedByKey)
	return value == kiteNamespaceManagedBy
}

// DeleteKiteUserResources removes the namespace resources owned by one deleted KiteUser.
// ctx controls the Kubernetes API calls made during cleanup.
// dynamicClient deletes NetworkPolicy and ResourceQuota objects after checking remaining KiteUser references.
// eventObj is the delete informer event object for the KiteUser resource.
// This function is used by the KiteUser delete event handler.
func DeleteKiteUserResources(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	user, err := kiteUserFromEventObject(eventObj)
	if err != nil {
		return err
	}

	if user.Spec.Namespace == "" {
		return fmt.Errorf("KiteUser %s has empty spec.namespace", user.GetName())
	}

	referenced, err := namespaceReferencedByKiteUser(ctx, dynamicClient, user.Spec.Namespace)
	if err != nil {
		return err
	}
	if referenced {
		log.Printf("kept KiteUser %s namespace resources in %s because another KiteUser still references the namespace", user.GetName(), user.Spec.Namespace)
		return nil
	}

	objects, err := userBaseResources(user)
	if err != nil {
		return err
	}

	for i := len(objects) - 1; i >= 0; i-- {
		if objects[i].GroupVersionKind().Kind == "Namespace" {
			continue
		}

		if err := deleteUserResource(ctx, dynamicClient, objects[i]); err != nil {
			return err
		}
	}

	log.Printf("deleted KiteUser %s namespace resources in %s", user.GetName(), user.Spec.Namespace)
	return nil
}

// kiteUserFromEventObject converts an informer event object into a KiteUser struct.
// eventObj can be an unstructured KiteUser or a DeletedFinalStateUnknown tombstone.
// The returned KiteUser contains metadata and spec fields used by the reconcile flow.
// This helper is used by add, update, and delete event handlers.
func kiteUserFromEventObject(eventObj interface{}) (*kitev1.KiteUser, error) {
	if tombstone, ok := eventObj.(cache.DeletedFinalStateUnknown); ok {
		eventObj = tombstone.Obj
	}

	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("KiteUser event object is not unstructured")
	}

	var user kitev1.KiteUser
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &user); err != nil {
		return nil, fmt.Errorf("failed to convert KiteUser %s: %w", resource.GetName(), err)
	}

	return &user, nil
}

// userBaseResources renders the namespace, network policies, and resource quota for a KiteUser.
// user provides spec.namespace and metadata.name values from the KiteUser CRD.
// The returned objects are applied in order so namespaced policies are created after the namespace.
// This helper is used by ReconcileKiteUser.
func userBaseResources(user *kitev1.KiteUser) ([]*unstructured.Unstructured, error) {
	namespaceObject, err := (&namespace.NamespaceData{
		Namespace: user.Spec.Namespace,
	}).Render()
	if err != nil {
		return nil, fmt.Errorf("failed to render namespace for KiteUser %s: %w", user.GetName(), err)
	}

	networkPolicyObjects, err := (&networkpolicy.NetworkPolicyData{
		Namespace: user.Spec.Namespace,
	}).RenderAll()
	if err != nil {
		return nil, fmt.Errorf("failed to render network policies for KiteUser %s: %w", user.GetName(), err)
	}

	quotaPolicyObjects, err := (&quotapolicy.QuotaPolicyData{
		Namespace: user.Spec.Namespace,
	}).RenderAll()
	if err != nil {
		return nil, fmt.Errorf("failed to render quota policies for KiteUser %s: %w", user.GetName(), err)
	}

	objects := make([]*unstructured.Unstructured, 0, 1+len(networkPolicyObjects)+len(quotaPolicyObjects))
	objects = append(objects, namespaceObject)
	objects = append(objects, networkPolicyObjects...)
	objects = append(objects, quotaPolicyObjects...)

	return objects, nil
}

// applyUserResource applies one rendered user-owned base resource with server-side apply.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient is used because controller renderers return unstructured Kubernetes objects.
// obj must be one of Namespace, NetworkPolicy, or ResourceQuota.
// This helper keeps KiteUser reconcile idempotent across repeated informer events.
func applyUserResource(ctx context.Context, dynamicClient dynamic.Interface, obj *unstructured.Unstructured) error {
	gvr, namespaced, err := userResourceGVR(obj)
	if err != nil {
		return err
	}

	data, err := json.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal %s/%s: %w", obj.GetKind(), obj.GetName(), err)
	}

	if namespaced {
		resource := dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
		if _, err := resource.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "kite-controller-user-reconciler",
			Force:        ptr.To(true),
		}); err != nil {
			if apierrors.IsNotFound(err) {
				if _, createErr := resource.Create(ctx, obj, metav1.CreateOptions{}); createErr != nil {
					return fmt.Errorf("failed to create %s %s/%s after apply NotFound: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), createErr)
				} else {
					return nil
				}
			}

			return fmt.Errorf("failed to apply %s %s/%s: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
		}

		return nil
	}

	resource := dynamicClient.Resource(gvr)
	if _, err := resource.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "kite-controller-user-reconciler",
		Force:        ptr.To(true),
	}); err != nil {
		if apierrors.IsNotFound(err) {
			if _, createErr := resource.Create(ctx, obj, metav1.CreateOptions{}); createErr != nil {
				return fmt.Errorf("failed to create %s %s/%s after apply NotFound: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), createErr)
			} else {
				return nil
			}
		}

		return fmt.Errorf("failed to apply %s %s/%s: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
	}

	return nil
}

// deleteUserResource deletes one rendered user-owned base resource.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient is used because controller renderers return unstructured Kubernetes objects.
// obj must be one of Namespace, NetworkPolicy, or ResourceQuota.
// This helper ignores NotFound errors so cleanup remains idempotent across repeated delete events.
func deleteUserResource(ctx context.Context, dynamicClient dynamic.Interface, obj *unstructured.Unstructured) error {
	gvr, namespaced, err := userResourceGVR(obj)
	if err != nil {
		return err
	}

	if namespaced {
		err = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	} else {
		err = dynamicClient.Resource(gvr).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	}

	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete %s %s/%s: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
	}

	return nil
}

// userResourceGVR returns the Kubernetes resource mapping for KiteUser base resources.
// obj provides the apiVersion and kind rendered from project YAML templates.
// The boolean return value is true when the resource must be applied inside obj.namespace.
// This helper avoids guessing plural names for resources such as NetworkPolicy.
func userResourceGVR(obj *unstructured.Unstructured) (schema.GroupVersionResource, bool, error) {
	switch obj.GroupVersionKind().Kind {
	case "Namespace":
		return namespaceGVR, false, nil
	case "NetworkPolicy":
		return networkPolicyGVR, true, nil
	case "ResourceQuota":
		return resourceQuotaGVR, true, nil
	default:
		return schema.GroupVersionResource{}, false, fmt.Errorf("unsupported KiteUser base resource kind %q", obj.GetKind())
	}
}
