package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"kite/internal/account"
	"kite/internal/auth"
	vmservice "kite/internal/vm"
)

const (
	levelOneFixedCPU    = 2
	levelOneFixedMemory = "4Gi"
	levelOneFixedDisk   = "20Gi"
)

type vmCreateRequest struct {
	Name         string `json:"name"`
	CPU          int    `json:"cpu"`
	Memory       string `json:"memory"`
	Image        string `json:"image"`
	Disk         any    `json:"disk" binding:"required"`
	DomainPrefix string `json:"domainPrefix"`
	SSHID        string `json:"sshId" binding:"required"`
	SSHPassword  string `json:"sshPassword" binding:"required"`
	PowerState   string `json:"powerState"`
}

type vmUpdateRequest struct {
	CPU          *int    `json:"cpu"`
	Memory       *string `json:"memory"`
	Image        *string `json:"image"`
	Disk         any     `json:"disk"`
	DomainPrefix *string `json:"domainPrefix"`
	SSHID        *string `json:"sshId"`
	SSHPassword  *string `json:"sshPassword"`
	PowerState   *string `json:"powerState"`
}

type vmPowerRequest struct {
	PowerState string `json:"powerState" binding:"required"`
}

// RegisterVirtualMachines attaches user VM routes to the versioned API router.
// api is the /api/v1 router group.
// deps provides auth and Kubernetes dependencies.
// Level 0 users are not allowed to use VM APIs.
// This function is used by RegisterV1 for frontend dashboard VM operations.
func RegisterVirtualMachines(api *gin.RouterGroup, deps Dependencies) {
	vms := api.Group("/vms", RequireAccessLevel(deps, auth.AccessLevelUser))
	vms.GET("", vmListHandler(deps))
	vms.POST("", vmCreateHandler(deps))
	vms.GET("/:name", vmGetHandler(deps))
	vms.PATCH("/:name", vmUpdateHandler(deps))
	vms.DELETE("/:name", vmDeleteHandler(deps))
	vms.POST("/:name/start", vmPowerHandler(deps, "On"))
	vms.POST("/:name/stop", vmPowerHandler(deps, "Off"))
}

// vmListHandler returns VMs in the authenticated user's namespace.
// deps provides Kubernetes and auth dependencies.
// The namespace is loaded from the current KiteUser rather than request input.
// This handler is used by the user dashboard VM table.
func vmListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		vms, err := vmServiceFromDependencies(deps).List(c.Request.Context(), user.Namespace)
		if err != nil {
			writeVMError(c, err, "failed to list virtual machines")
			return
		}

		c.JSON(http.StatusOK, gin.H{"vms": vms})
	}
}

// vmCreateHandler creates a KiteVirtualMachine CRD in the current user's namespace.
// deps provides Kubernetes and auth dependencies.
// The request body contains frontend VM form fields.
// Level 0 users cannot create VMs, and level 1 users always receive the fixed entry-level spec.
// This handler does not call KubeVirt directly; kite-controller performs provisioning.
func vmCreateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}
		if user.AccessLevel < int64(auth.AccessLevelUser) {
			c.JSON(http.StatusForbidden, gin.H{"message": "VM creation requires access level 1 or higher"})
			return
		}

		var req vmCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}
		applyAccessLevelCreateLimits(user.AccessLevel, &req)

		created, err := vmServiceFromDependencies(deps).Create(c.Request.Context(), vmservice.CreateRequest{
			Name:         req.Name,
			Namespace:    user.Namespace,
			CPU:          req.CPU,
			Memory:       req.Memory,
			Image:        req.Image,
			Disk:         vmservice.NormalizeDisk(req.Disk),
			DomainPrefix: req.DomainPrefix,
			SSHID:        req.SSHID,
			SSHPassword:  req.SSHPassword,
			PowerState:   req.PowerState,
		})
		if err != nil {
			writeVMError(c, err, "failed to create virtual machine")
			return
		}

		c.JSON(http.StatusCreated, gin.H{"vm": created})
	}
}

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

// vmGetHandler returns one VM from the authenticated user's namespace.
// deps provides Kubernetes and auth dependencies.
// The route parameter is metadata.name of the KiteVirtualMachine CRD.
// This handler is used by VM detail pages.
func vmGetHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		vm, err := vmServiceFromDependencies(deps).Get(c.Request.Context(), user.Namespace, c.Param("name"))
		if err != nil {
			writeVMError(c, err, "failed to read virtual machine")
			return
		}

		c.JSON(http.StatusOK, gin.H{"vm": vm})
	}
}

// vmUpdateHandler patches mutable VM desired-state fields.
// deps provides Kubernetes and auth dependencies.
// The route parameter is metadata.name in the current user's namespace.
// Level 1 users cannot change the fixed CPU, memory, or disk limits.
// This handler is used for general VM edits and direct powerState patches.
func vmUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		var req vmUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}
		applyAccessLevelUpdateLimits(user.AccessLevel, &req)

		disk := normalizeOptionalDisk(req.Disk)
		vm, err := vmServiceFromDependencies(deps).Update(c.Request.Context(), user.Namespace, c.Param("name"), vmservice.UpdateRequest{
			CPU:          req.CPU,
			Memory:       req.Memory,
			Image:        req.Image,
			Disk:         disk,
			DomainPrefix: req.DomainPrefix,
			SSHID:        req.SSHID,
			SSHPassword:  req.SSHPassword,
			PowerState:   req.PowerState,
		})
		if err != nil {
			writeVMError(c, err, "failed to update virtual machine")
			return
		}

		c.JSON(http.StatusOK, gin.H{"vm": vm})
	}
}

// vmDeleteHandler marks one VM for controller-managed deletion.
// deps provides Kubernetes and auth dependencies.
// The route parameter is metadata.name in the current user's namespace.
// This handler sets spec.delete=true so finalizer cleanup owns actual deletion.
func vmDeleteHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		vm, err := vmServiceFromDependencies(deps).MarkDelete(c.Request.Context(), user.Namespace, c.Param("name"))
		if err != nil {
			writeVMError(c, err, "failed to delete virtual machine")
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "virtual machine delete requested", "vm": vm})
	}
}

// vmPowerHandler returns a handler that sets VM spec.powerState.
// deps provides Kubernetes and auth dependencies.
// powerState is either On or Off and is written to the KiteVirtualMachine spec.
// This handler supports simple start and stop frontend actions.
func vmPowerHandler(deps Dependencies, powerState string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		vm, err := vmServiceFromDependencies(deps).Update(c.Request.Context(), user.Namespace, c.Param("name"), vmservice.UpdateRequest{
			PowerState: &powerState,
		})
		if err != nil {
			writeVMError(c, err, "failed to update virtual machine power state")
			return
		}

		c.JSON(http.StatusOK, gin.H{"vm": vm})
	}
}

// currentUser returns the KiteUser matching the authenticated JWT subject.
// c provides claims set by RequireAccessLevel.
// deps provides Kubernetes access through account service.
// The boolean return is false after an HTTP error response has been written.
func currentUser(c *gin.Context, deps Dependencies) (account.PublicUser, bool) {
	claims, ok := currentClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "access token is required"})
		return account.PublicUser{}, false
	}

	accountService, ok := accountServiceFromDependencies(c, deps)
	if !ok {
		return account.PublicUser{}, false
	}

	userObject, found, err := accountService.FindByUsername(c.Request.Context(), claims.Subject)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read kite users"})
		return account.PublicUser{}, false
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "current user was not found"})
		return account.PublicUser{}, false
	}

	user, err := accountService.Get(c.Request.Context(), userObject.GetName())
	if err != nil {
		writeAccountError(c, err, "failed to read current user")
		return account.PublicUser{}, false
	}

	return user, true
}

// vmServiceFromDependencies creates a VM service for one request.
// deps provides the dynamic Kubernetes client.
// The returned service reads and writes KiteVirtualMachine CRDs.
func vmServiceFromDependencies(deps Dependencies) *vmservice.Service {
	return vmservice.NewService(deps.DynamicClient, deps.Config.PasswordSalt)
}

// normalizeOptionalDisk converts PATCH disk input to an optional string pointer.
// value is nil when the client omitted disk.
// The returned pointer is nil when no update is requested.
func normalizeOptionalDisk(value any) *string {
	if value == nil {
		return nil
	}

	normalized := vmservice.NormalizeDisk(value)
	return &normalized
}

// writeVMError maps VM service and Kubernetes errors to HTTP responses.
// c is the active Gin request context.
// err is returned by internal/vm or Kubernetes store code.
// fallbackMessage is used for unexpected internal errors.
func writeVMError(c *gin.Context, err error, fallbackMessage string) {
	if apierrors.IsNotFound(err) || vmservice.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"message": "virtual machine was not found"})
		return
	}
	if message := vmservice.ConflictMessage(err); message != "" {
		c.JSON(http.StatusConflict, gin.H{"message": message})
		return
	}
	if kind, ok := vmservice.RequestErrorKind(err); ok {
		switch kind {
		case vmservice.ErrorKindInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": fallbackMessage})
		}
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"message": fallbackMessage})
}
