package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KiteUser represents the cluster-scoped KiteUser custom resource.
// This type maps to custom/kite-user-crd.yaml and is used when converting
// unstructured KiteUser objects into Go structs.
type KiteUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KiteUserSpec `json:"spec,omitempty"`
}

// KiteUserSpec contains the spec fields defined by custom/kite-user-crd.yaml.
type KiteUserSpec struct {
	Username     string `json:"username,omitempty"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	ProfileImage string `json:"profile_image,omitempty"`
	AccessLevel  int    `json:"access_level,omitempty"`
}

// KiteUserList represents a Kubernetes list response for KiteUser resources.
type KiteUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KiteUser `json:"items"`
}

// KiteVirtualMachine represents the namespaced KiteVirtualMachine custom resource.
// This type maps to custom/kite-machine-crd.yaml and is used by the controller
// when converting unstructured KiteVirtualMachine objects into Go structs.
type KiteVirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KiteVirtualMachineSpec   `json:"spec,omitempty"`
	Status KiteVirtualMachineStatus `json:"status,omitempty"`
}

// KiteVirtualMachineSpec contains the user-provided spec fields defined by
// custom/kite-machine-crd.yaml.
type KiteVirtualMachineSpec struct {
	CPU    int    `json:"cpu,omitempty"`
	Memory string `json:"memory"`
	Image  string `json:"image,omitempty"`
	Disk   string `json:"disk,omitempty"`
}

// KiteVirtualMachineStatus contains the controller-managed status fields
// defined by custom/kite-machine-crd.yaml.
type KiteVirtualMachineStatus struct {
	Phase string `json:"phase,omitempty"`
}

// KiteVirtualMachineList represents a Kubernetes list response for
// KiteVirtualMachine resources.
type KiteVirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KiteVirtualMachine `json:"items"`
}
