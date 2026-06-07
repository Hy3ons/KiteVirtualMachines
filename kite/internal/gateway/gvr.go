package gateway

import "k8s.io/apimachinery/pkg/runtime/schema"

var kiteVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
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
