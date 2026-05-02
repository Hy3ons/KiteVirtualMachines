package store

import "k8s.io/apimachinery/pkg/runtime/schema"

var kiteUserGVR = schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kiteusers",
}

var kiteVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}
