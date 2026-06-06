package apps

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
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
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"

	kite "kite/api/v1"
	"kite/internal/kube"
	cloudinituserdata "kite/internal/render/cloud-init-userdata"
	datavolume "kite/internal/render/data-volume"
	"kite/internal/render/ingress"
	kubevirtmachine "kite/internal/render/kubevirt-machine"
	vmservice "kite/internal/render/vm-service"
	"kite/internal/sshkey"
)

var kiteVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}

var kubeVirtVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "kubevirt.io",
	Version:  "v1",
	Resource: "virtualmachines",
}

var dataVolumeGVR = schema.GroupVersionResource{
	Group:    "cdi.kubevirt.io",
	Version:  "v1beta1",
	Resource: "datavolumes",
}

var secretGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "secrets",
}

var serviceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}

var ingressGVR = schema.GroupVersionResource{
	Group:    "networking.k8s.io",
	Version:  "v1",
	Resource: "ingresses",
}

var configMapGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "configmaps",
}

const (
	kiteVirtualMachineCleanupFinalizer = "kite.anacnu.com/kite-vm-cleanup"
	kiteVirtualMachineStatusManager    = "kite-controller-vm-status"
	kiteVirtualMachineApplyManager     = "kite-controller-vm-reconciler"
	kiteVMPhaseProvisioning            = "Provisioning"
	kiteVMPhaseRunning                 = "Running"
	kiteVMPhaseStopped                 = "Stopped"
	kiteVMPhaseReady                   = "Ready"
	kiteVMPhaseDeleting                = "Deleting"
	kiteVMPhaseFailed                  = "Failed"
	kiteVMConditionReady               = "VirtualMachineReady"
	kiteVMReasonReconciled             = "Reconciled"
	kiteVMReasonDeleting               = "Deleting"
	kiteVMReasonFailed                 = "ReconcileFailed"
	kiteGlobalConfigNamespace          = "kite"
	kiteGlobalConfigName               = "kite-runtime-config"
	kiteGlobalBaseDomainKey            = "baseDomain"
	kiteSecretTypeLabel                = "kite.anacnu.com/kite-secret-type"
	kiteVMSSHKeySecretType             = "vm-ssh-key"
	vmSSHPrivateKeyName                = "id_rsa"
	vmSSHPublicKeyName                 = "id_rsa.pub"
)

// RunKiteVirtualMachineReconciler creates and runs the KiteVirtualMachine informer loop.
// clientManager provides the dynamic Kubernetes client used to watch KiteVirtualMachine resources.
// stopCh stops the informer when the controller process is shutting down.
// This function is expected to run in a goroutine from cmd/kite-controller/main.go.
func RunKiteVirtualMachineReconciler(clientManager *kube.ClientManager, stopCh <-chan struct{}) {
	if clientManager == nil || clientManager.DynamicClient == nil {
		log.Printf("KiteVirtualMachine reconciler requires a dynamic Kubernetes client")
		return
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(clientManager.DynamicClient, time.Second*30)
	informer := factory.ForResource(kiteVirtualMachineGVR).Informer()
	RegisterKiteVirtualMachineReconciler(informer, clientManager.DynamicClient)

	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		log.Printf("failed to sync KiteVirtualMachine informer cache")
		return
	}

	<-stopCh
}

// RegisterKiteVirtualMachineReconciler attaches KiteVirtualMachine event handlers to an informer.
// informer watches namespaced KiteVirtualMachine custom resources from this controller app.
// dynamicClient is used to reconcile the desired KiteVirtualMachine state against KubeVirt resources.
// This function is used by RunKiteVirtualMachineReconciler during controller startup.
func RegisterKiteVirtualMachineReconciler(informer cache.SharedIndexInformer, dynamicClient dynamic.Interface) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := ReconcileKiteVirtualMachine(context.Background(), dynamicClient, obj); err != nil {
				log.Printf("failed to reconcile KiteVirtualMachine add event: %v", err)
			}
		},

		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldResource, ok := oldObj.(metav1.Object)
			if !ok {
				log.Printf("old KiteVirtualMachine update event object is not metav1.Object")
				return
			}

			newResource, ok := newObj.(metav1.Object)
			if !ok {
				log.Printf("new KiteVirtualMachine update event object is not metav1.Object")
				return
			}

			if oldResource.GetGeneration() == newResource.GetGeneration() && !kiteVirtualMachineNeedsSameGenerationReconcile(newObj) {
				return
			}

			if err := ReconcileKiteVirtualMachine(context.Background(), dynamicClient, newObj); err != nil {
				log.Printf("failed to reconcile KiteVirtualMachine update event: %v", err)
			}
		},

		DeleteFunc: func(obj interface{}) {
			vm, err := kiteVirtualMachineFromEventObject(obj)
			if err != nil {
				log.Printf("failed to read KiteVirtualMachine delete event: %v", err)
				return
			}

			if err := deleteKubeVirtVirtualMachine(context.Background(), dynamicClient, vm.Namespace, vm.Name); err != nil {
				log.Printf("failed to delete KubeVirt VirtualMachine for removed KiteVirtualMachine %s/%s: %v", vm.Namespace, vm.Name, err)
				return
			}
			if err := deleteKiteVirtualMachineOwnedResources(context.Background(), dynamicClient, vm.Namespace, vm.Name); err != nil {
				log.Printf("failed to delete owned resources for removed KiteVirtualMachine %s/%s: %v", vm.Namespace, vm.Name, err)
				return
			}

			log.Printf("KiteVirtualMachine deleted: %s/%s", vm.Namespace, vm.Name)
		},
	})
}

// ReconcileKiteVirtualMachine reconciles one KiteVirtualMachine event object.
// ctx controls Kubernetes API calls made during reconcile.
// dynamicClient reads and deletes KiteVirtualMachine and KubeVirt VirtualMachine resources.
// eventObj is the informer object for the KiteVirtualMachine being reconciled.
// A nil error means this reconcile pass completed or had no work to do.
func ReconcileKiteVirtualMachine(ctx context.Context, dynamicClient dynamic.Interface, eventObj interface{}) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic client is required for KiteVirtualMachine reconcile")
	}

	resource, err := kiteVirtualMachineResourceFromEventObject(eventObj)
	if err != nil {
		return err
	}

	vm, err := kiteVirtualMachineFromEventObject(eventObj)
	if err != nil {
		return err
	}

	if vm.Namespace == "" {
		log.Printf("KiteVirtualMachine %q has empty namespace; skipping reconcile", vm.Name)
		return nil
	}

	if vm.DeletionTimestamp != nil {
		return reconcileKiteVirtualMachineDeletion(ctx, dynamicClient, resource, vm)
	}

	if !hasKiteVirtualMachineFinalizer(vm.Finalizers) {
		if err := updateKiteVirtualMachineFinalizers(ctx, dynamicClient, resource, append(vm.Finalizers, kiteVirtualMachineCleanupFinalizer)); err != nil {
			return err
		}

		resource.SetFinalizers(append(resource.GetFinalizers(), kiteVirtualMachineCleanupFinalizer))
	}

	if !vm.Spec.Delete {
		return reconcileKiteVirtualMachineDesiredState(ctx, dynamicClient, vm)
	}

	if err := updateKiteVirtualMachineStatus(ctx, dynamicClient, vm, kiteVMPhaseDeleting, currentPowerStateFromVM(vm), "", vm.Status.NodeName, metav1.ConditionFalse, kiteVMReasonDeleting, "delete intent accepted; removing VM-owned resources"); err != nil {
		return err
	}

	exists, err := kubeVirtVirtualMachineExists(ctx, dynamicClient, vm.Namespace, vm.Name)
	if err != nil {
		return err
	}

	if exists {
		if err := deleteKubeVirtVirtualMachine(ctx, dynamicClient, vm.Namespace, vm.Name); err != nil {
			return err
		}
		if err := deleteKiteVirtualMachineOwnedResources(ctx, dynamicClient, vm.Namespace, vm.Name); err != nil {
			return err
		}

		log.Printf("delete intent accepted for KiteVirtualMachine %s/%s; waiting for KubeVirt VirtualMachine removal", vm.Namespace, vm.Name)
		return nil
	}

	if err := deleteKiteVirtualMachineOwnedResources(ctx, dynamicClient, vm.Namespace, vm.Name); err != nil {
		return err
	}

	if err := deleteKiteVirtualMachineCRD(ctx, dynamicClient, vm.Namespace, vm.Name); err != nil {
		return err
	}

	log.Printf("deleted KiteVirtualMachine CRD after KubeVirt VirtualMachine was absent: %s/%s", vm.Namespace, vm.Name)
	return nil
}

// reconcileKiteVirtualMachineDesiredState applies the real resources for one KiteVirtualMachine.
// ctx controls Kubernetes API calls made during resource apply and status updates.
// dynamicClient applies DataVolume, Secret, KubeVirt VM, Service, and optional Ingress resources.
// vm provides the desired spec that should be reflected in the cluster.
func reconcileKiteVirtualMachineDesiredState(ctx context.Context, dynamicClient dynamic.Interface, vm *kite.KiteVirtualMachine) error {
	if err := validateKiteVirtualMachineSpec(vm); err != nil {
		if statusErr := updateKiteVirtualMachineStatus(ctx, dynamicClient, vm, kiteVMPhaseFailed, currentPowerStateFromVM(vm), "", vm.Status.NodeName, metav1.ConditionFalse, kiteVMReasonFailed, err.Error()); statusErr != nil {
			return fmt.Errorf("%w; failed to update failed status: %v", err, statusErr)
		}

		return err
	}

	domain, err := kiteVirtualMachineDomain(ctx, dynamicClient, vm)
	if err != nil {
		return err
	}

	keyPair, err := ensureKiteVirtualMachineSSHKeySecret(ctx, dynamicClient, vm)
	if err != nil {
		if statusErr := updateKiteVirtualMachineStatus(ctx, dynamicClient, vm, kiteVMPhaseFailed, currentPowerStateFromVM(vm), domain, vm.Status.NodeName, metav1.ConditionFalse, kiteVMReasonFailed, err.Error()); statusErr != nil {
			return fmt.Errorf("%w; failed to update failed status: %v", err, statusErr)
		}
		return err
	}

	objects, err := kiteVirtualMachineDesiredObjects(vm, domain, keyPair.PublicKey)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		if err := applyKiteVirtualMachineObject(ctx, dynamicClient, obj); err != nil {
			if statusErr := updateKiteVirtualMachineStatus(ctx, dynamicClient, vm, kiteVMPhaseFailed, currentPowerStateFromVM(vm), domain, vm.Status.NodeName, metav1.ConditionFalse, kiteVMReasonFailed, err.Error()); statusErr != nil {
				return fmt.Errorf("%w; failed to update failed status: %v", err, statusErr)
			}

			return err
		}
	}

	if err := updateKiteVirtualMachineStatus(ctx, dynamicClient, vm, kiteVMPhaseProvisioning, currentPowerStateFromVM(vm), domain, vm.Status.NodeName, metav1.ConditionFalse, kiteVMReasonReconciled, "VM resources are applied and waiting for KubeVirt readiness"); err != nil {
		return err
	}

	log.Printf("reconciled KiteVirtualMachine desired resources: %s/%s", vm.Namespace, vm.Name)
	return nil
}

// validateKiteVirtualMachineSpec checks the minimum fields needed to create VM resources.
// vm is the KiteVirtualMachine being reconciled.
// A nil error means the controller can render and apply resources from the spec.
func validateKiteVirtualMachineSpec(vm *kite.KiteVirtualMachine) error {
	missing := make([]string, 0)
	if vm.Name == "" {
		missing = append(missing, "metadata.name")
	}
	if vm.Namespace == "" {
		missing = append(missing, "metadata.namespace")
	}
	if vm.Spec.CPU <= 0 {
		missing = append(missing, "spec.cpu")
	}
	if strings.TrimSpace(vm.Spec.Memory) == "" {
		missing = append(missing, "spec.memory")
	}
	if strings.TrimSpace(vm.Spec.Image) == "" {
		missing = append(missing, "spec.image")
	}
	if strings.TrimSpace(vm.Spec.Disk) == "" {
		missing = append(missing, "spec.disk")
	}
	if strings.TrimSpace(vm.Spec.SSHID) == "" {
		missing = append(missing, "spec.sshId")
	}
	if strings.TrimSpace(vm.Spec.SSHPassword) == "" {
		missing = append(missing, "spec.sshPassword")
	}
	if powerState := strings.TrimSpace(vm.Spec.PowerState); powerState != "" && powerState != "On" && powerState != "Off" {
		return fmt.Errorf("spec.powerState must be On or Off")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}

	return nil
}

// kiteVirtualMachineDesiredObjects renders every Kubernetes object owned by one KiteVirtualMachine.
// vm provides the desired VM spec.
// domain is the optional full domain name used to render Ingress.
// The returned objects are applied in dependency order by the VM reconciler.
func kiteVirtualMachineDesiredObjects(vm *kite.KiteVirtualMachine, domain string, publicKey string) ([]*unstructured.Unstructured, error) {
	dataVolume, err := (&datavolume.DataVolumeData{
		VmName:    vm.Name,
		Namespace: vm.Namespace,
		VmImage:   datavolume.VmImage(vm.Spec.Image),
		Storage:   vm.Spec.Disk,
	}).Render()
	if err != nil {
		return nil, fmt.Errorf("failed to render DataVolume: %w", err)
	}

	cloudInit, err := (&cloudinituserdata.Ubuntu2204CloudInit{
		VmName:       vm.Name,
		Namespace:    vm.Namespace,
		Id:           vm.Spec.SSHID,
		Password:     vm.Spec.SSHPassword,
		SSHPublicKey: publicKey,
	}).Render()
	if err != nil {
		return nil, fmt.Errorf("failed to render cloud-init Secret: %w", err)
	}

	kubeVirtVM, err := (&kubevirtmachine.KubevirtMachineData{
		VmName:      vm.Name,
		Namespace:   vm.Namespace,
		Memory:      vm.Spec.Memory,
		CPU:         strconv.Itoa(vm.Spec.CPU),
		RunStrategy: kubeVirtRunStrategyForKiteVM(vm),
	}).Render()
	if err != nil {
		return nil, fmt.Errorf("failed to render KubeVirt VirtualMachine: %w", err)
	}

	serviceObjects, err := (&vmservice.ServiceData{
		VmName:    vm.Name,
		Namespace: vm.Namespace,
	}).RenderAll()
	if err != nil {
		return nil, fmt.Errorf("failed to render VM Services: %w", err)
	}

	objects := make([]*unstructured.Unstructured, 0, 3+len(serviceObjects)+1)
	objects = append(objects, dataVolume, cloudInit, kubeVirtVM)
	objects = append(objects, serviceObjects...)

	if domain != "" {
		ingressObject, err := (&ingress.IngressData{
			VmName:     vm.Name,
			Namespace:  vm.Namespace,
			DomainName: domain,
		}).Render()
		if err != nil {
			return nil, fmt.Errorf("failed to render Ingress: %w", err)
		}
		objects = append(objects, ingressObject)
	}

	return objects, nil
}

// vmShouldRun converts Kite power intent into a boolean running intent.
// vm provides spec.powerState from the user.
// The returned value defaults empty powerState to false so VM creation does not auto-start unexpectedly.
func vmShouldRun(vm *kite.KiteVirtualMachine) bool {
	return strings.TrimSpace(vm.Spec.PowerState) == "On"
}

// kubeVirtRunStrategyForKiteVM converts Kite power intent into KubeVirt spec.runStrategy.
// vm provides spec.powerState from the user.
// The returned value uses runStrategy instead of deprecated spec.running.
func kubeVirtRunStrategyForKiteVM(vm *kite.KiteVirtualMachine) string {
	if vmShouldRun(vm) {
		return "Always"
	}

	return "Halted"
}

// currentPowerStateFromVM returns the desired power state string stored in VM status.
// vm provides spec.powerState.
// The returned value defaults empty powerState to Off.
func currentPowerStateFromVM(vm *kite.KiteVirtualMachine) string {
	if vmShouldRun(vm) {
		return "On"
	}

	return "Off"
}

// ensureKiteVirtualMachineSSHKeySecret returns the stable VM SSH key pair Secret.
// ctx controls Kubernetes API calls.
// dynamicClient reads or creates the core/v1 Secret in the KiteVirtualMachine namespace.
// vm identifies the VM whose cloud-init public key and host-agent private key must match.
func ensureKiteVirtualMachineSSHKeySecret(ctx context.Context, dynamicClient dynamic.Interface, vm *kite.KiteVirtualMachine) (sshkey.KeyPair, error) {
	name := sshKeySecretName(vm.Name)
	current, err := dynamicClient.Resource(secretGVR).Namespace(vm.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return keyPairFromSecret(current)
	}
	if !apierrors.IsNotFound(err) {
		return sshkey.KeyPair{}, fmt.Errorf("failed to read SSH key Secret %s/%s: %w", vm.Namespace, name, err)
	}

	keyPair, err := sshkey.GenerateRSA(2048)
	if err != nil {
		return sshkey.KeyPair{}, fmt.Errorf("failed to generate SSH key pair for %s/%s: %w", vm.Namespace, vm.Name, err)
	}

	secret := newKiteVirtualMachineSSHKeySecret(vm, keyPair)
	if _, err := dynamicClient.Resource(secretGVR).Namespace(vm.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return sshkey.KeyPair{}, fmt.Errorf("failed to create SSH key Secret %s/%s: %w", vm.Namespace, name, err)
		}
	}
	return keyPair, nil
}

// keyPairFromSecret decodes the VM SSH key Secret managed by kite-controller.
// secret is a core/v1 Secret with data.id_rsa and data.id_rsa.pub.
// The returned pair is used by cloud-init rendering and host-agent account reconcile.
func keyPairFromSecret(secret *unstructured.Unstructured) (sshkey.KeyPair, error) {
	data, _, _ := unstructured.NestedStringMap(secret.Object, "data")
	privateKey, err := base64.StdEncoding.DecodeString(data[vmSSHPrivateKeyName])
	if err != nil {
		return sshkey.KeyPair{}, fmt.Errorf("failed to decode %s in Secret %s/%s: %w", vmSSHPrivateKeyName, secret.GetNamespace(), secret.GetName(), err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(data[vmSSHPublicKeyName])
	if err != nil {
		return sshkey.KeyPair{}, fmt.Errorf("failed to decode %s in Secret %s/%s: %w", vmSSHPublicKeyName, secret.GetNamespace(), secret.GetName(), err)
	}

	if strings.TrimSpace(string(privateKey)) == "" || strings.TrimSpace(string(publicKey)) == "" {
		return sshkey.KeyPair{}, fmt.Errorf("SSH key Secret %s/%s is missing required key data", secret.GetNamespace(), secret.GetName())
	}

	return sshkey.KeyPair{
		PrivateKey: string(privateKey),
		PublicKey:  strings.TrimSpace(string(publicKey)),
	}, nil
}

// newKiteVirtualMachineSSHKeySecret renders the Secret that stores one VM SSH key pair.
// vm identifies the owning KiteVirtualMachine.
// keyPair contains the private key for kite-host-agent and the public key for cloud-init.
func newKiteVirtualMachineSSHKeySecret(vm *kite.KiteVirtualMachine, keyPair sshkey.KeyPair) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      sshKeySecretName(vm.Name),
				"namespace": vm.Namespace,
				"labels": map[string]any{
					kiteManagedByLabel:   kiteControllerLabel,
					kiteSecretTypeLabel:  kiteVMSSHKeySecretType,
					kiteVMNameLabel:      vm.Name,
					kiteVMNamespaceLabel: vm.Namespace,
				},
			},
			"type": "Opaque",
			"data": map[string]any{
				vmSSHPrivateKeyName: base64.StdEncoding.EncodeToString([]byte(keyPair.PrivateKey)),
				vmSSHPublicKeyName:  base64.StdEncoding.EncodeToString([]byte(keyPair.PublicKey)),
			},
		},
	}
}

// kiteVirtualMachineDomain returns the optional full domain name for a VM.
// ctx controls the ConfigMap read.
// dynamicClient reads kite/kite-runtime-config when spec.domainPrefix is set.
// vm provides spec.domainPrefix and metadata used to compose the domain.
func kiteVirtualMachineDomain(ctx context.Context, dynamicClient dynamic.Interface, vm *kite.KiteVirtualMachine) (string, error) {
	prefix := strings.TrimSpace(vm.Spec.DomainPrefix)
	if prefix == "" {
		return "", nil
	}

	config, err := dynamicClient.Resource(configMapGVR).Namespace(kiteGlobalConfigNamespace).Get(ctx, kiteGlobalConfigName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read global Kite config: %w", err)
	}

	baseDomain, _, _ := unstructured.NestedString(config.Object, "data", kiteGlobalBaseDomainKey)
	baseDomain = strings.Trim(strings.TrimSpace(baseDomain), ".")
	if baseDomain == "" {
		return "", nil
	}

	return strings.Trim(prefix, ".") + "." + baseDomain, nil
}

// applyKiteVirtualMachineObject applies one rendered VM-owned Kubernetes object.
// ctx controls the Kubernetes API request lifetime.
// dynamicClient writes the unstructured object through the matching resource endpoint.
// obj must be one of DataVolume, Secret, VirtualMachine, Service, or Ingress.
func applyKiteVirtualMachineObject(ctx context.Context, dynamicClient dynamic.Interface, obj *unstructured.Unstructured) error {
	gvr, err := kiteVirtualMachineObjectGVR(obj)
	if err != nil {
		return err
	}

	data, err := json.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal %s/%s: %w", obj.GetKind(), obj.GetName(), err)
	}

	_, err = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: kiteVirtualMachineApplyManager,
		Force:        ptr.To(true),
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("%s resource API is not installed in this cluster; %s", obj.GetKind(), missingKiteVirtualMachineDependencyMessage(obj.GetKind()))
		}

		return fmt.Errorf("failed to apply %s %s/%s: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
	}

	return nil
}

// missingKiteVirtualMachineDependencyMessage explains why a rendered dependency cannot be applied.
// kind is the Kubernetes resource kind that failed with a NotFound API error.
// The returned message is written into KiteVirtualMachine status and controller logs.
func missingKiteVirtualMachineDependencyMessage(kind string) string {
	switch kind {
	case "DataVolume":
		return "install CDI before creating KiteVirtualMachine disks"
	case "VirtualMachine":
		return "install KubeVirt before creating KiteVirtualMachine resources"
	case "Ingress":
		return "install networking.k8s.io Ingress support before enabling VM domains"
	default:
		return "install the required Kubernetes API before reconciling this resource"
	}
}

// kiteVirtualMachineObjectGVR maps rendered VM resource kinds to explicit GVRs.
// obj provides apiVersion and kind from a renderer.
// The returned GVR is used by server-side apply and delete helpers.
func kiteVirtualMachineObjectGVR(obj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	switch obj.GroupVersionKind().Kind {
	case "DataVolume":
		return dataVolumeGVR, nil
	case "Secret":
		return secretGVR, nil
	case "VirtualMachine":
		return kubeVirtVirtualMachineGVR, nil
	case "Service":
		return serviceGVR, nil
	case "Ingress":
		return ingressGVR, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unsupported KiteVirtualMachine resource kind %q", obj.GetKind())
	}
}

// reconcileKiteVirtualMachineDeletion performs finalizer cleanup for a deleting KiteVirtualMachine.
// ctx controls Kubernetes API calls made during deletion cleanup.
// dynamicClient deletes the matching KubeVirt VirtualMachine and updates the KiteVirtualMachine finalizer.
// resource is the current unstructured KiteVirtualMachine object with resourceVersion.
// vm is the typed view used for namespace, name, and finalizer checks.
func reconcileKiteVirtualMachineDeletion(ctx context.Context, dynamicClient dynamic.Interface, resource *unstructured.Unstructured, vm *kite.KiteVirtualMachine) error {
	if !hasKiteVirtualMachineFinalizer(vm.Finalizers) {
		return nil
	}

	exists, err := kubeVirtVirtualMachineExists(ctx, dynamicClient, vm.Namespace, vm.Name)
	if err != nil {
		return err
	}

	if exists {
		if err := deleteKubeVirtVirtualMachine(ctx, dynamicClient, vm.Namespace, vm.Name); err != nil {
			return err
		}
		if err := deleteKiteVirtualMachineOwnedResources(ctx, dynamicClient, vm.Namespace, vm.Name); err != nil {
			return err
		}

		log.Printf("KiteVirtualMachine %s/%s is deleting; waiting for KubeVirt VirtualMachine removal", vm.Namespace, vm.Name)
		return nil
	}

	if err := deleteKiteVirtualMachineOwnedResources(ctx, dynamicClient, vm.Namespace, vm.Name); err != nil {
		return err
	}

	if err := updateKiteVirtualMachineFinalizers(ctx, dynamicClient, resource, removeKiteVirtualMachineFinalizer(vm.Finalizers)); err != nil {
		return err
	}

	log.Printf("removed KiteVirtualMachine cleanup finalizer after KubeVirt VirtualMachine was absent: %s/%s", vm.Namespace, vm.Name)
	return nil
}

// kiteVirtualMachineResourceFromEventObject extracts the unstructured object from an informer event.
// eventObj may be a direct unstructured object or a DeletedFinalStateUnknown tombstone.
// The returned object keeps resourceVersion and metadata for finalizer updates.
func kiteVirtualMachineResourceFromEventObject(eventObj interface{}) (*unstructured.Unstructured, error) {
	resource, ok := eventObj.(*unstructured.Unstructured)
	if !ok {
		if tombstone, tombstoneOK := eventObj.(cache.DeletedFinalStateUnknown); tombstoneOK {
			resource, ok = tombstone.Obj.(*unstructured.Unstructured)
		}
	}
	if !ok {
		return nil, fmt.Errorf("KiteVirtualMachine event object is not unstructured")
	}

	return resource, nil
}

// kiteVirtualMachineFromEventObject converts an informer event object into a KiteVirtualMachine.
// eventObj may be a normal unstructured object or a DeletedFinalStateUnknown tombstone from the cache.
// The returned struct is used by reconcile and delete handlers to find namespace, name, and spec intent.
func kiteVirtualMachineFromEventObject(eventObj interface{}) (*kite.KiteVirtualMachine, error) {
	resource, err := kiteVirtualMachineResourceFromEventObject(eventObj)
	if err != nil {
		return nil, err
	}

	var vm kite.KiteVirtualMachine
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &vm); err != nil {
		return nil, err
	}

	return &vm, nil
}

// kiteVirtualMachineNeedsSameGenerationReconcile reports whether an update should run despite unchanged generation.
// eventObj is the new KiteVirtualMachine object from an update or resync event.
// A true result allows delete intent and deletionTimestamp updates to finish cleanup.
func kiteVirtualMachineNeedsSameGenerationReconcile(eventObj interface{}) bool {
	vm, err := kiteVirtualMachineFromEventObject(eventObj)
	if err != nil {
		log.Printf("failed to inspect KiteVirtualMachine same-generation reconcile need: %v", err)
		return false
	}

	return vm.Spec.Delete || vm.DeletionTimestamp != nil
}

// kubeVirtVirtualMachineExists checks whether the real KubeVirt VirtualMachine currently exists.
// ctx controls the Kubernetes API get request.
// dynamicClient reads kubevirt.io/v1 virtualmachines in the KiteVirtualMachine namespace.
// The boolean return is false when Kubernetes returns NotFound.
func kubeVirtVirtualMachineExists(ctx context.Context, dynamicClient dynamic.Interface, namespace string, name string) (bool, error) {
	_, err := dynamicClient.Resource(kubeVirtVirtualMachineGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

// deleteKubeVirtVirtualMachine deletes the real KubeVirt VirtualMachine if it exists.
// ctx controls the Kubernetes API delete request.
// dynamicClient deletes kubevirt.io/v1 virtualmachines in the KiteVirtualMachine namespace.
// NotFound is treated as success so direct user or cluster-side cleanup does not leave reconcile stuck.
func deleteKubeVirtVirtualMachine(ctx context.Context, dynamicClient dynamic.Interface, namespace string, name string) error {
	return deleteOwnedNamespacedResource(ctx, dynamicClient, kubeVirtVirtualMachineGVR, namespace, name, namespace, name)
}

// deleteKiteVirtualMachineOwnedResources deletes non-VM Kubernetes objects owned by one KiteVirtualMachine.
// ctx controls Kubernetes API delete requests.
// dynamicClient deletes Ingress, Service, Secret, and DataVolume resources in namespace.
// namespace and name identify the KiteVirtualMachine whose rendered child resources should be removed.
func deleteKiteVirtualMachineOwnedResources(ctx context.Context, dynamicClient dynamic.Interface, namespace string, name string) error {
	resources := []struct {
		gvr  schema.GroupVersionResource
		name string
	}{
		{gvr: ingressGVR, name: name},
		{gvr: serviceGVR, name: "vps-access-" + name},
		{gvr: serviceGVR, name: "vps-web-" + name},
		{gvr: secretGVR, name: sshKeySecretName(name)},
		{gvr: secretGVR, name: name + "-cloud-init-userdata"},
		{gvr: dataVolumeGVR, name: name + "-disk"},
	}

	for _, resource := range resources {
		if err := deleteOwnedNamespacedResource(ctx, dynamicClient, resource.gvr, namespace, resource.name, namespace, name); err != nil {
			return fmt.Errorf("failed to delete owned resource %s/%s in %s: %w", resource.gvr.Resource, resource.name, namespace, err)
		}
	}

	return nil
}

// deleteOwnedNamespacedResource deletes one namespaced resource only when Kite owner labels match.
// ctx controls Kubernetes get and delete calls.
// dynamicClient reads the current object before deleting it.
// gvr, namespace, and name identify the candidate child resource.
// ownerNamespace and ownerName identify the KiteVirtualMachine that is allowed to own the resource.
func deleteOwnedNamespacedResource(ctx context.Context, dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace string, name string, ownerNamespace string, ownerName string) error {
	current, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if !resourceOwnedByKiteVirtualMachine(current, ownerNamespace, ownerName) {
		log.Printf("skipped deleting %s/%s in %s because Kite owner labels do not match %s/%s", gvr.Resource, name, namespace, ownerNamespace, ownerName)
		return nil
	}

	err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}

	return err
}

// resourceOwnedByKiteVirtualMachine checks Kite ownership labels before destructive cleanup.
// obj is a Kubernetes resource that may have been created by kite-controller.
// ownerNamespace and ownerName identify the KiteVirtualMachine being deleted.
// The returned value is true only for resources explicitly labeled as owned by that VM.
func resourceOwnedByKiteVirtualMachine(obj *unstructured.Unstructured, ownerNamespace string, ownerName string) bool {
	labels := obj.GetLabels()
	return labels[kiteManagedByLabel] == kiteControllerLabel &&
		labels[kiteVMNamespaceLabel] == ownerNamespace &&
		labels[kiteVMNameLabel] == ownerName
}

// updateKiteVirtualMachineStatus writes controller-observed state back to a KiteVirtualMachine CRD.
// ctx controls Kubernetes API get and status update calls.
// dynamicClient updates anacnu.com/v1 kitevirtualmachines through the status subresource.
// vm identifies the CRD and provides the metadata generation used as observedGeneration.
func updateKiteVirtualMachineStatus(ctx context.Context, dynamicClient dynamic.Interface, vm *kite.KiteVirtualMachine, phase string, powerState string, domain string, nodeName string, conditionStatus metav1.ConditionStatus, reason string, message string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(vm.Namespace).Get(ctx, vm.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read KiteVirtualMachine %s/%s before status update: %w", vm.Namespace, vm.Name, err)
		}

		observedGeneration := current.GetGeneration()
		if kiteVirtualMachineStatusMatches(current, observedGeneration, phase, powerState, domain, nodeName, conditionStatus, reason, message) {
			return nil
		}

		next := current.DeepCopy()
		if err := unstructured.SetNestedField(next.Object, phase, "status", "phase"); err != nil {
			return err
		}
		if err := unstructured.SetNestedField(next.Object, powerState, "status", "currentPowerState"); err != nil {
			return err
		}
		if err := unstructured.SetNestedField(next.Object, observedGeneration, "status", "observedGeneration"); err != nil {
			return err
		}
		if domain != "" {
			if err := unstructured.SetNestedField(next.Object, domain, "status", "domain"); err != nil {
				return err
			}
		} else {
			unstructured.RemoveNestedField(next.Object, "status", "domain")
		}
		if nodeName != "" {
			if err := unstructured.SetNestedField(next.Object, nodeName, "status", "nodeName"); err != nil {
				return err
			}
		} else {
			unstructured.RemoveNestedField(next.Object, "status", "nodeName")
		}
		resourceNames := kiteVirtualMachineResourceNames(vm.Name)
		for field, value := range resourceNames {
			if err := unstructured.SetNestedField(next.Object, value, "status", field); err != nil {
				return err
			}
		}

		conditions, _, _ := unstructured.NestedSlice(next.Object, "status", "conditions")
		conditions = replaceKiteVirtualMachineCondition(conditions, kiteVirtualMachineCondition(observedGeneration, conditionStatus, reason, message))
		if err := unstructured.SetNestedSlice(next.Object, conditions, "status", "conditions"); err != nil {
			return err
		}

		_, err = dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(vm.Namespace).UpdateStatus(ctx, next, metav1.UpdateOptions{
			FieldManager: kiteVirtualMachineStatusManager,
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update KiteVirtualMachine %s/%s status: %w", vm.Namespace, vm.Name, err)
	}

	return nil
}

// kiteVirtualMachineStatusMatches checks whether a status update would be a no-op.
// current is the latest KiteVirtualMachine object from the API server.
// The remaining parameters are the status values the controller wants to write.
// A true return lets reconcile avoid noisy status updates.
func kiteVirtualMachineStatusMatches(current *unstructured.Unstructured, observedGeneration int64, phase string, powerState string, domain string, nodeName string, conditionStatus metav1.ConditionStatus, reason string, message string) bool {
	currentPhase, _, _ := unstructured.NestedString(current.Object, "status", "phase")
	currentPowerState, _, _ := unstructured.NestedString(current.Object, "status", "currentPowerState")
	currentObservedGeneration, _, _ := unstructured.NestedInt64(current.Object, "status", "observedGeneration")
	currentDomain, _, _ := unstructured.NestedString(current.Object, "status", "domain")
	currentNodeName, _, _ := unstructured.NestedString(current.Object, "status", "nodeName")
	currentCondition := findKiteVirtualMachineCondition(current.Object)
	for field, value := range kiteVirtualMachineResourceNames(current.GetName()) {
		currentValue, _, _ := unstructured.NestedString(current.Object, "status", field)
		if currentValue != value {
			return false
		}
	}

	return currentPhase == phase &&
		currentPowerState == powerState &&
		currentObservedGeneration == observedGeneration &&
		currentDomain == domain &&
		currentNodeName == nodeName &&
		machineStringValue(currentCondition["status"]) == string(conditionStatus) &&
		machineStringValue(currentCondition["reason"]) == reason &&
		machineStringValue(currentCondition["message"]) == message
}

// kiteVirtualMachineCondition creates the Ready condition map used in KiteVirtualMachine status.
// observedGeneration is the metadata generation processed by the controller.
// conditionStatus, reason, and message describe the current reconcile outcome.
// The returned map is stored in status.conditions.
func kiteVirtualMachineCondition(observedGeneration int64, conditionStatus metav1.ConditionStatus, reason string, message string) map[string]any {
	return map[string]any{
		"type":               kiteVMConditionReady,
		"status":             string(conditionStatus),
		"lastTransitionTime": metav1.Now().Format(time.RFC3339),
		"reason":             reason,
		"message":            message,
		"observedGeneration": observedGeneration,
	}
}

// replaceKiteVirtualMachineCondition replaces the controller-owned Ready condition.
// conditions is the current status.conditions slice from an unstructured object.
// next is the condition value the controller wants to store.
// The returned slice preserves unrelated condition types.
func replaceKiteVirtualMachineCondition(conditions []any, next map[string]any) []any {
	replaced := false
	output := make([]any, 0, len(conditions)+1)
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if machineStringValue(condition["type"]) == kiteVMConditionReady {
			output = append(output, next)
			replaced = true
			continue
		}
		output = append(output, condition)
	}
	if !replaced {
		output = append(output, next)
	}

	return output
}

// findKiteVirtualMachineCondition returns the controller-owned Ready condition from an object.
// object is the unstructured KiteVirtualMachine object map.
// The returned map is empty when the condition has not been written yet.
func findKiteVirtualMachineCondition(object map[string]any) map[string]any {
	conditions, _, _ := unstructured.NestedSlice(object, "status", "conditions")
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if machineStringValue(condition["type"]) == kiteVMConditionReady {
			return condition
		}
	}

	return map[string]any{}
}

// machineStringValue converts an unstructured field to a string for status comparison.
// value is usually a condition field read from a map.
// The returned string is empty when the field is absent or not a string.
func machineStringValue(value any) string {
	text, _ := value.(string)
	return text
}

// kiteVirtualMachineResourceNames returns stable child resource names for one KiteVM.
// vmName is metadata.name from the KiteVirtualMachine.
// The returned map is written to status so agents can find controller-owned resources.
func kiteVirtualMachineResourceNames(vmName string) map[string]string {
	return map[string]string{
		"sshKeySecretName":    sshKeySecretName(vmName),
		"cloudInitSecretName": vmName + "-cloud-init-userdata",
		"serviceName":         "vps-access-" + vmName,
		"dataVolumeName":      vmName + "-disk",
	}
}

// sshKeySecretName returns the stable Secret name for a VM SSH key pair.
// vmName is metadata.name from the KiteVirtualMachine.
// The returned name is used by kite-controller and kite-host-agent.
func sshKeySecretName(vmName string) string {
	return vmName + "-ssh-key"
}

// updateKiteVirtualMachineFinalizers updates finalizers on one KiteVirtualMachine CRD.
// ctx controls the Kubernetes API update request.
// dynamicClient writes anacnu.com/v1 kitevirtualmachines in the resource namespace.
// resource is copied so the informer cache object is not mutated unexpectedly.
// finalizers is the complete replacement finalizer list.
func updateKiteVirtualMachineFinalizers(ctx context.Context, dynamicClient dynamic.Interface, resource *unstructured.Unstructured, finalizers []string) error {
	next := resource.DeepCopy()
	next.SetFinalizers(finalizers)

	_, err := dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(next.GetNamespace()).Update(ctx, next, metav1.UpdateOptions{})
	return err
}

// hasKiteVirtualMachineFinalizer reports whether the cleanup finalizer is present.
// finalizers is the metadata.finalizers list from a KiteVirtualMachine object.
// The returned value is used before adding or removing finalizers during reconcile.
func hasKiteVirtualMachineFinalizer(finalizers []string) bool {
	for _, finalizer := range finalizers {
		if finalizer == kiteVirtualMachineCleanupFinalizer {
			return true
		}
	}

	return false
}

// removeKiteVirtualMachineFinalizer returns a finalizer list without the cleanup finalizer.
// finalizers is the metadata.finalizers list from a KiteVirtualMachine object.
// The returned slice is used as the replacement list when cleanup is complete.
func removeKiteVirtualMachineFinalizer(finalizers []string) []string {
	next := make([]string, 0, len(finalizers))
	for _, finalizer := range finalizers {
		if finalizer != kiteVirtualMachineCleanupFinalizer {
			next = append(next, finalizer)
		}
	}

	return next
}

// deleteKiteVirtualMachineCRD deletes the KiteVirtualMachine CRD after real resources are gone.
// ctx controls the Kubernetes API delete request.
// dynamicClient deletes anacnu.com/v1 kitevirtualmachines in the same namespace.
// NotFound is treated as success because another reconcile pass may have already removed the CRD.
func deleteKiteVirtualMachineCRD(ctx context.Context, dynamicClient dynamic.Interface, namespace string, name string) error {
	err := dynamicClient.Resource(kiteVirtualMachineGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}

	return err
}
