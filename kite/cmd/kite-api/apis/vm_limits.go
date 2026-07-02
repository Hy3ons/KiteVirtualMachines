package apis

import "kite/internal/auth"

const (
	levelOneFixedCPU    = 2
	levelOneFixedMemory = "4Gi"
	levelOneFixedDisk   = "20Gi"
	levelOneVMQuota     = 3
)

// applyAccessLevelCreateLimits enforces VM create-time resource limits from the authenticated user.
// accessLevel is copied from the current KiteUser CRD.
// req is the parsed HTTP create body and is modified before it reaches the VM service.
// This function is used by vmCreateHandler to keep frontend limits authoritative on the API server.
func applyAccessLevelCreateLimits(accessLevel int64, req *vmCreateRequest) {
	if accessLevel != int64(auth.AccessLevelUser) {
		return
	}

	req.CPU = levelOneFixedCPU
	req.Memory = levelOneFixedMemory
	req.Disk = levelOneFixedDisk
}

// applyAccessLevelUpdateLimits enforces fixed resource updates for level 1 users.
// accessLevel is copied from the current KiteUser CRD.
// req is the parsed PATCH body and is modified only when resource fields are present.
// This function is used by vmUpdateHandler to prevent direct API calls from raising level 1 VM specs.
func applyAccessLevelUpdateLimits(accessLevel int64, req *vmUpdateRequest) {
	if accessLevel != int64(auth.AccessLevelUser) {
		return
	}

	if req.CPU != nil {
		cpu := levelOneFixedCPU
		req.CPU = &cpu
	}
	if req.Memory != nil {
		memory := levelOneFixedMemory
		req.Memory = &memory
	}
	if req.Disk != nil {
		req.Disk = levelOneFixedDisk
	}
}
