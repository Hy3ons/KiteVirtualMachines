package store

import "k8s.io/apimachinery/pkg/runtime/schema"

var kiteUserGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kiteusers",
}

var kiteVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}

var kiteVirtualMachineOfferGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachineoffers",
}
