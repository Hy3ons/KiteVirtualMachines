package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	kitev1 "kite/api/v1"
	"kite/internal/hostaccount"
	"kite/internal/kube"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const (
	resyncPeriod              = time.Second * 30
	localGCPeriod             = time.Second * 60
	crdMissingSweepThreshold  = 5
	kiteManagedByLabel        = "kite.anacnu.com/managed-by"
	kiteVMNameLabel           = "kite.anacnu.com/kite-vm-name"
	kiteVMNamespaceLabel      = "kite.anacnu.com/kite-vm-namespace"
	kiteSecretTypeLabel       = "kite.anacnu.com/kite-secret-type"
	kiteControllerLabel       = "kite-controller"
	kiteVMSSHKeySecretType    = "vm-ssh-key"
	sshServiceNamePrefix      = "vps-access-"
	defaultHostRoot           = "/host"
	kiteHostAgentServiceName  = "kite-host-agent"
	kiteHostAgentStartupError = "kite-host-agent requires a Kubernetes dynamic client"
	vmSSHPrivateKeyName       = "id_rsa"
)

var (
	kiteVirtualMachineGVR = schema.GroupVersionResource{
		Group:    "anacnu.com",
		Version:  "v1",
		Resource: "kitevirtualmachines",
	}
	secretGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}
)

func main() {
	clientManager, err := kube.GetClientManager()
	if err != nil {
		log.Fatalf("failed to connect to Kubernetes cluster: %v", err)
	}
	if clientManager.DynamicClient == nil {
		log.Fatalf(kiteHostAgentStartupError)
	}

	hostRoot := strings.TrimSpace(os.Getenv("KITE_HOST_AGENT_HOST_ROOT"))
	if hostRoot == "" {
		hostRoot = defaultHostRoot
	}
	nodeName := strings.TrimSpace(os.Getenv("KITE_NODE_NAME"))

	manager := hostaccount.NewManager(hostRoot)
	stopCh := make(chan struct{})
	runHostAgent(clientManager.DynamicClient, manager, nodeName, stopCh)
}

// runHostAgent starts VM and Secret informers plus local metadata garbage collection.
// dynamicClient watches KiteVM and Secret resources.
// manager reconciles host Linux accounts under the host filesystem.
// nodeName is this DaemonSet pod's Kubernetes node name.
func runHostAgent(dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string, stopCh <-chan struct{}) {
	if dynamicClient == nil {
		log.Printf(kiteHostAgentStartupError)
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, resyncPeriod)
	vmInformer := factory.ForResource(kiteVirtualMachineGVR).Informer()
	secretInformer := factory.ForResource(secretGVR).Informer()

	registerKiteVMHandler(vmInformer, dynamicClient, manager, nodeName)
	registerSecretHandler(secretInformer, dynamicClient, manager, nodeName)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, vmInformer.HasSynced, secretInformer.HasSynced) {
		log.Printf("failed to sync %s informer caches", kiteHostAgentServiceName)
		return
	}

	go runLocalGC(dynamicClient, manager, nodeName, stopCh)
	<-stopCh
}

// registerKiteVMHandler attaches host account reconcile handlers to KiteVirtualMachine events.
// informer watches all KiteVM CRDs.
// dynamicClient reads VM SSH key Secrets.
// manager applies or deletes host account state.
func registerKiteVMHandler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := reconcileKiteVMEvent(context.Background(), dynamicClient, manager, nodeName, obj, false); err != nil {
				log.Printf("failed to reconcile Kite host account add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := reconcileKiteVMEvent(context.Background(), dynamicClient, manager, nodeName, newObj, false); err != nil {
				log.Printf("failed to reconcile Kite host account update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := reconcileKiteVMEvent(context.Background(), dynamicClient, manager, nodeName, obj, true); err != nil {
				log.Printf("failed to reconcile Kite host account delete event: %v", err)
			}
		},
	})
}

// registerSecretHandler attaches reconcile handlers to VM SSH key Secret events.
// informer watches core/v1 Secrets.
// dynamicClient reads the owning KiteVM through labels on the Secret.
// manager refreshes host private keys when the Secret changes.
func registerSecretHandler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := reconcileSecretEvent(context.Background(), dynamicClient, manager, nodeName, obj); err != nil {
				log.Printf("failed to reconcile Kite host Secret add event: %v", err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if err := reconcileSecretEvent(context.Background(), dynamicClient, manager, nodeName, newObj); err != nil {
				log.Printf("failed to reconcile Kite host Secret update event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := reconcileSecretEvent(context.Background(), dynamicClient, manager, nodeName, obj); err != nil {
				log.Printf("failed to reconcile Kite host Secret delete event: %v", err)
			}
		},
	})
}

// reconcileKiteVMEvent reconciles or deletes one host account from a KiteVM event.
// ctx controls Kubernetes and host command calls.
// dynamicClient reads the matching SSH key Secret.
// manager applies host account changes.
// deleted forces cleanup for informer delete events.
func reconcileKiteVMEvent(ctx context.Context, dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string, eventObj interface{}, deleted bool) error {
	vm, err := kiteVirtualMachineFromEvent(eventObj)
	if err != nil {
		return err
	}
	if vm.Namespace == "" || vm.Name == "" || strings.TrimSpace(vm.Spec.SSHID) == "" {
		return nil
	}

	if deleted || vm.Spec.Delete || !accountShouldHandleVM(vm, nodeName) {
		return manager.Delete(ctx, vm.Spec.SSHID, vm.Namespace, vm.Name)
	}

	return ensureAccountForVM(ctx, dynamicClient, manager, nodeName, vm)
}

// reconcileSecretEvent reconciles the host account for the KiteVM that owns one SSH key Secret.
// ctx controls Kubernetes and host command calls.
// dynamicClient reads the owning KiteVM.
// manager applies host account state when the Secret contains a private key.
func reconcileSecretEvent(ctx context.Context, dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string, eventObj interface{}) error {
	secret, err := secretFromEvent(eventObj)
	if err != nil {
		return err
	}
	if !isVMSSHKeySecret(secret) {
		return nil
	}

	labels := secret.GetLabels()
	vmNamespace := labels[kiteVMNamespaceLabel]
	vmName := labels[kiteVMNameLabel]
	if vmNamespace == "" || vmName == "" {
		return nil
	}

	vmObject, err := dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(vmNamespace).Get(ctx, vmName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read KiteVirtualMachine %s/%s for Secret reconcile: %w", vmNamespace, vmName, err)
	}

	return reconcileKiteVMEvent(ctx, dynamicClient, manager, nodeName, vmObject, false)
}

// ensureAccountForVM creates or updates the host account for one live KiteVM.
// ctx controls Kubernetes Secret reads and host command execution.
// dynamicClient reads the VM SSH key Secret.
// manager writes host account state under /home and /var/lib/kite/accounts.
func ensureAccountForVM(ctx context.Context, dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string, vm *kitev1.KiteVirtualMachine) error {
	secretName := sshKeySecretNameForVM(vm)
	secret, err := dynamicClient.Resource(secretGVR).Namespace(vm.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read SSH key Secret %s/%s: %w", vm.Namespace, secretName, err)
	}

	privateKey, err := privateKeyFromSecret(secret)
	if err != nil {
		return err
	}

	return manager.Ensure(ctx, hostaccount.DesiredAccount{
		Username:         vm.Spec.SSHID,
		Password:         vm.Spec.SSHPassword,
		VMNamespace:      vm.Namespace,
		VMName:           vm.Name,
		NodeName:         nodeName,
		SSHKeySecretName: secretName,
		PrivateKey:       privateKey,
		ServiceName:      serviceNameForVM(vm),
		ServiceNamespace: vm.Namespace,
		VMUser:           vm.Spec.SSHID,
	})
}

// runLocalGC periodically compares local Kite-owned accounts with current cluster state.
// dynamicClient lists KiteVM objects and reads key Secrets.
// manager lists and deletes Kite-owned host accounts.
// nodeName is used to keep only accounts assigned to this node.
func runLocalGC(dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string, stopCh <-chan struct{}) {
	ticker := time.NewTicker(localGCPeriod)
	defer ticker.Stop()

	missingCRDCount := 0
	for {
		select {
		case <-ticker.C:
			nextMissingCount, err := reconcileLocalAccounts(context.Background(), dynamicClient, manager, nodeName, missingCRDCount)
			if err != nil {
				log.Printf("failed to run Kite host account GC: %v", err)
			}
			missingCRDCount = nextMissingCount
		case <-stopCh:
			return
		}
	}
}

// reconcileLocalAccounts sweeps local metadata that no longer has a matching cluster desired state.
// ctx controls Kubernetes reads and host delete commands.
// dynamicClient lists KiteVMs and reads Secrets.
// previousMissingCRDCount prevents deleting all accounts during a transient CRD discovery failure.
func reconcileLocalAccounts(ctx context.Context, dynamicClient dynamic.Interface, manager *hostaccount.Manager, nodeName string, previousMissingCRDCount int) (int, error) {
	owners, err := manager.ListOwners()
	if err != nil {
		return previousMissingCRDCount, err
	}
	if len(owners) == 0 {
		return 0, nil
	}

	list, err := dynamicClient.Resource(kiteVirtualMachineGVR).List(ctx, metav1.ListOptions{})
	if apierrors.IsNotFound(err) {
		missingCount := previousMissingCRDCount + 1
		if missingCount >= crdMissingSweepThreshold {
			for _, owner := range owners {
				if err := manager.Delete(ctx, owner.Username, owner.VMNamespace, owner.VMName); err != nil {
					return missingCount, err
				}
			}
		}
		return missingCount, nil
	}
	if err != nil {
		return previousMissingCRDCount, err
	}

	vms := make(map[string]*kitev1.KiteVirtualMachine, len(list.Items))
	for i := range list.Items {
		vm, err := kiteVirtualMachineFromUnstructured(&list.Items[i])
		if err != nil {
			return 0, err
		}
		vms[vmKey(vm.Namespace, vm.Name)] = vm
	}

	for _, owner := range owners {
		vm := vms[vmKey(owner.VMNamespace, owner.VMName)]
		if shouldDeleteLocalAccount(ctx, dynamicClient, nodeName, owner, vm) {
			if err := manager.Delete(ctx, owner.Username, owner.VMNamespace, owner.VMName); err != nil {
				return 0, err
			}
		}
	}

	return 0, nil
}

// shouldDeleteLocalAccount reports whether one local metadata entry is orphaned.
// ctx controls the Secret existence check.
// dynamicClient reads the VM key Secret.
// owner is the local account metadata and vm is the current cluster VM, if any.
func shouldDeleteLocalAccount(ctx context.Context, dynamicClient dynamic.Interface, nodeName string, owner hostaccount.OwnerMetadata, vm *kitev1.KiteVirtualMachine) bool {
	if vm == nil || vm.Spec.Delete {
		return true
	}
	if vm.Spec.SSHID != owner.Username {
		return true
	}
	if !accountShouldHandleVM(vm, nodeName) {
		return true
	}

	secretName := strings.TrimSpace(owner.SSHKeySecretName)
	if secretName == "" {
		secretName = sshKeySecretNameForVM(vm)
	}
	_, err := dynamicClient.Resource(secretGVR).Namespace(owner.VMNamespace).Get(ctx, secretName, metav1.GetOptions{})
	return apierrors.IsNotFound(err)
}

// accountShouldHandleVM checks whether this host-agent should reconcile one KiteVM.
// vm provides status.nodeName written by the controller when KubeVirt reports placement.
// nodeName is the DaemonSet pod's own spec.nodeName; empty is allowed for single-node local testing.
func accountShouldHandleVM(vm *kitev1.KiteVirtualMachine, nodeName string) bool {
	assignedNode := strings.TrimSpace(vm.Status.NodeName)
	if assignedNode == "" || strings.TrimSpace(nodeName) == "" {
		return true
	}

	return assignedNode == strings.TrimSpace(nodeName)
}

// privateKeyFromSecret decodes data.id_rsa from a VM SSH key Secret.
// secret is the Secret created by kite-controller for a KiteVM.
// The returned private key is written into the host user's ~/.ssh/id_rsa.
func privateKeyFromSecret(secret *unstructured.Unstructured) (string, error) {
	data, _, _ := unstructured.NestedStringMap(secret.Object, "data")
	encoded := data[vmSSHPrivateKeyName]
	privateKey, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode %s in Secret %s/%s: %w", vmSSHPrivateKeyName, secret.GetNamespace(), secret.GetName(), err)
	}
	if strings.TrimSpace(string(privateKey)) == "" {
		return "", fmt.Errorf("Secret %s/%s has empty %s", secret.GetNamespace(), secret.GetName(), vmSSHPrivateKeyName)
	}
	return string(privateKey), nil
}

// isVMSSHKeySecret checks whether a Secret is managed by Kite as a VM SSH key.
// secret is a core/v1 Secret event object.
// A true return means the labels point to a KiteVirtualMachine owner.
func isVMSSHKeySecret(secret *unstructured.Unstructured) bool {
	labels := secret.GetLabels()
	return labels[kiteManagedByLabel] == kiteControllerLabel &&
		labels[kiteSecretTypeLabel] == kiteVMSSHKeySecretType
}

// sshKeySecretNameForVM returns the Secret name used for one VM key pair.
// vm may already have status.sshKeySecretName written by the controller.
// The returned fallback is stable when status is not populated yet.
func sshKeySecretNameForVM(vm *kitev1.KiteVirtualMachine) string {
	if strings.TrimSpace(vm.Status.SSHKeySecretName) != "" {
		return strings.TrimSpace(vm.Status.SSHKeySecretName)
	}
	return vm.Name + "-ssh-key"
}

// serviceNameForVM returns the Service DNS name prefix used by the custom login shell.
// vm may already have status.serviceName written by the controller.
// The returned fallback matches the controller's fixed vps-access naming rule.
func serviceNameForVM(vm *kitev1.KiteVirtualMachine) string {
	if strings.TrimSpace(vm.Status.ServiceName) != "" {
		return strings.TrimSpace(vm.Status.ServiceName)
	}
	return sshServiceNamePrefix + vm.Name
}

// vmKey creates a stable namespace/name key for map lookups.
// namespace and name identify a KiteVirtualMachine.
// The returned string is used only inside kite-host-agent GC.
func vmKey(namespace string, name string) string {
	return namespace + "/" + name
}

// kiteVirtualMachineFromEvent extracts a typed KiteVirtualMachine from an informer event.
// eventObj may be a direct unstructured object or a delete tombstone.
// The returned VM provides spec and status for account reconcile.
func kiteVirtualMachineFromEvent(eventObj interface{}) (*kitev1.KiteVirtualMachine, error) {
	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		if tombstone, tombstoneOK := eventObj.(cache.DeletedFinalStateUnknown); tombstoneOK {
			resource, ok = tombstone.Obj.(*unstructured.Unstructured)
		}
	}
	if !ok {
		return nil, fmt.Errorf("KiteVirtualMachine event object is not unstructured")
	}

	return kiteVirtualMachineFromUnstructured(resource)
}

// kiteVirtualMachineFromUnstructured converts a KiteVM unstructured object into the local API struct.
// resource is the current anacnu.com/v1 KiteVirtualMachine object.
// The returned struct is used by informer and GC reconcile paths.
func kiteVirtualMachineFromUnstructured(resource *unstructured.Unstructured) (*kitev1.KiteVirtualMachine, error) {
	var vm kitev1.KiteVirtualMachine
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
		return nil, err
	}
	return &vm, nil
}

// secretFromEvent extracts an unstructured Secret from an informer event.
// eventObj may be a direct object or a delete tombstone.
// The returned object is inspected for Kite VM ownership labels.
func secretFromEvent(eventObj interface{}) (*unstructured.Unstructured, error) {
	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		if tombstone, tombstoneOK := eventObj.(cache.DeletedFinalStateUnknown); tombstoneOK {
			resource, ok = tombstone.Obj.(*unstructured.Unstructured)
		}
	}
	if !ok {
		return nil, fmt.Errorf("Secret event object is not unstructured")
	}

	return resource, nil
}
