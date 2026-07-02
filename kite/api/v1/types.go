package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KiteUser represents the cluster-scoped KiteUser custom resource.
// This type maps to build/kite/crds.yaml and is used when converting
// unstructured KiteUser objects into Go structs.
type KiteUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KiteUserSpec   `json:"spec,omitempty"`
	Status KiteUserStatus `json:"status,omitempty"`
}

// KiteUserSpec contains the KiteUser spec fields defined by build/kite/crds.yaml.
type KiteUserSpec struct {
	Username     string `json:"username,omitempty"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	ProfileImage string `json:"profile_image,omitempty"`
	AccessLevel  int    `json:"access_level,omitempty"`
}

// KiteUserStatus contains controller-managed status fields for KiteUser.
// Phase summarizes whether the user's namespace resources are ready or failed.
// ObservedGeneration records the metadata generation processed by the controller.
// ObservedNamespace records the spec.namespace last reconciled by the controller.
// Message gives a short human-readable result for kubectl describe.
// Conditions stores detailed reconcile state such as NamespaceReady.
type KiteUserStatus struct {
	Phase              string             `json:"phase,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	ObservedNamespace  string             `json:"observedNamespace,omitempty"`
	Message            string             `json:"message,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// KiteUserList represents a Kubernetes list response for KiteUser resources.
type KiteUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KiteUser `json:"items"`
}

// KiteVirtualMachine represents the namespaced KiteVirtualMachine custom resource.
// This type maps to build/kite/crds.yaml and is used by the controller
// when converting unstructured KiteVirtualMachine objects into Go structs.
type KiteVirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KiteVirtualMachineSpec   `json:"spec,omitempty"`
	Status KiteVirtualMachineStatus `json:"status,omitempty"`
}

// KiteVirtualMachineSpec contains the user-provided spec fields defined by
// build/kite/crds.yaml.
type KiteVirtualMachineSpec struct {
	CPU             int    `json:"cpu,omitempty"`
	Memory          string `json:"memory"`
	Image           string `json:"image,omitempty"`
	Disk            string `json:"disk,omitempty"`
	PowerState      string `json:"powerState,omitempty"`
	DomainPrefix    string `json:"domainPrefix,omitempty"`
	SSHID           string `json:"sshId,omitempty"`
	SSHPasswordHash string `json:"sshPasswordHash,omitempty"`
	Delete          bool   `json:"delete,omitempty"`
}

// KiteVirtualMachineStatus contains the controller-managed status fields
// defined by build/kite/crds.yaml.
type KiteVirtualMachineStatus struct {
	Phase               string             `json:"phase,omitempty"`
	CurrentPowerState   string             `json:"currentPowerState,omitempty"`
	ObservedGeneration  int64              `json:"observedGeneration,omitempty"`
	Domain              string             `json:"domain,omitempty"`
	NodeName            string             `json:"nodeName,omitempty"`
	SSHKeySecretName    string             `json:"sshKeySecretName,omitempty"`
	CloudInitSecretName string             `json:"cloudInitSecretName,omitempty"`
	ServiceName         string             `json:"serviceName,omitempty"`
	DataVolumeName      string             `json:"dataVolumeName,omitempty"`
	DataVolumePhase     string             `json:"dataVolumePhase,omitempty"`
	DataVolumeProgress  string             `json:"dataVolumeProgress,omitempty"`
	DataVolumeMessage   string             `json:"dataVolumeMessage,omitempty"`
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
}

// KiteVirtualMachineList represents a Kubernetes list response for
// KiteVirtualMachine resources.
type KiteVirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KiteVirtualMachine `json:"items"`
}

// KiteVirtualMachineOffer represents an admin-provided VM spec allocation.
// This type maps to build/kite/crds.yaml and is used when converting
// unstructured offer objects into typed controller or API values.
type KiteVirtualMachineOffer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KiteVirtualMachineOfferSpec   `json:"spec,omitempty"`
	Status KiteVirtualMachineOfferStatus `json:"status,omitempty"`
}

// KiteVirtualMachineOfferSpec contains the admin-provided VM capacity fields.
// The user fills only identity and login fields when claiming the offer.
type KiteVirtualMachineOfferSpec struct {
	CPU       int         `json:"cpu,omitempty"`
	Memory    string      `json:"memory,omitempty"`
	Disk      string      `json:"disk,omitempty"`
	Image     string      `json:"image,omitempty"`
	ExpiresAt metav1.Time `json:"expiresAt,omitempty"`
	CreatedBy string      `json:"createdBy,omitempty"`
}

// KiteVirtualMachineOfferStatus contains claim state for one VM offer.
// Claimed offers are deleted after the corresponding KiteVirtualMachine is created.
type KiteVirtualMachineOfferStatus struct {
	Phase              string `json:"phase,omitempty"`
	ClaimedBy          string `json:"claimedBy,omitempty"`
	Message            string `json:"message,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
}

// KiteVirtualMachineOfferList represents a Kubernetes list response for VM offers.
type KiteVirtualMachineOfferList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KiteVirtualMachineOffer `json:"items"`
}
