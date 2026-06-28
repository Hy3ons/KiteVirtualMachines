package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kite/internal/auth"
	vmservice "kite/internal/vm"
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
	vms.POST("/:name/console-ticket", vmConsoleTicketHandler(deps))

	// Console WebSocket requests authenticate with a short-lived ticket because
	// browser WebSocket constructors cannot attach the normal Authorization header.
	api.GET("/vms/:name/console", vmConsoleHandler(deps))
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
