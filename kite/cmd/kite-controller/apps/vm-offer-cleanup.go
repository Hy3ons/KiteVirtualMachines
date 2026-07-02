package apps

import (
	"context"
	"log"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"kite/internal/kube"
)

const virtualMachineOfferCleanupPeriod = time.Hour

var kiteVirtualMachineOfferGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachineoffers",
}

// RunKiteVirtualMachineOfferCleanup periodically removes expired VM offers.
// clientManager provides the dynamic Kubernetes client used to list and delete offers.
// stopCh stops the loop when the controller process is shutting down.
// This function is expected to run in a goroutine from cmd/kite-controller/main.go.
func RunKiteVirtualMachineOfferCleanup(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KiteVirtualMachineOffer cleanup requires a dynamic Kubernetes client")
		return
	}

	ticker := time.NewTicker(virtualMachineOfferCleanupPeriod)
	defer ticker.Stop()

	if err := DeleteExpiredKiteVirtualMachineOffers(context.Background(), clientManager.DynamicClient, time.Now().UTC()); err != nil {
		log.Printf("failed to delete expired VM offers: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := DeleteExpiredKiteVirtualMachineOffers(context.Background(), clientManager.DynamicClient, time.Now().UTC()); err != nil {
				log.Printf("failed to delete expired VM offers: %v", err)
			}
		case <-stopCh:
			return
		}
	}
}

// ctx controls Kubernetes API calls.
// dynamicClient lists and deletes KiteVirtualMachineOffer CRDs.
// now is injected by tests and compared against spec.expiresAt.
func DeleteExpiredKiteVirtualMachineOffers(ctx context.Context, dynamicClient dynamic.Interface, now time.Time) error {
	list, err := dynamicClient.Resource(kiteVirtualMachineOfferGVR).List(ctx, metav1.ListOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for i := range list.Items {
		offer := &list.Items[i]
		if !kiteVirtualMachineOfferExpired(offer, now) {
			continue
		}
		if err := dynamicClient.Resource(kiteVirtualMachineOfferGVR).Namespace(offer.GetNamespace()).Delete(ctx, offer.GetName(), metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// kiteVirtualMachineOfferExpired checks whether an offer should be cleaned up.
// offer is one KiteVirtualMachineOffer CRD object from the list call.
// now is the controller's current UTC time.
// Malformed expiresAt values are treated as expired so broken offers do not remain forever.
func kiteVirtualMachineOfferExpired(offer *unstructured.Unstructured, now time.Time) bool {
	expiresAtValue, _, _ := unstructured.NestedString(offer.Object, "spec", "expiresAt")
	expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(expiresAtValue))
	return err != nil || !expiresAt.After(now)
}
