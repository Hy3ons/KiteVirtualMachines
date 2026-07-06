package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kite/internal/account"
	"kite/internal/auth"
	"kite/internal/platform"
	vmservice "kite/internal/vm"
)

type accessLevelUpdateRequest struct {
	AccessLevel int `json:"access_level" binding:"required"`
}

type domainUpdateRequest struct {
	BaseDomain string `json:"baseDomain" binding:"required"`
}

type certUpdateRequest struct {
	TLSCert string `json:"tlsCert" binding:"required"`
	TLSKey  string `json:"tlsKey" binding:"required"`
}

type httpsUpdateRequest struct {
	ForceHTTPS bool `json:"forceHttps"`
}

type adminContactUpdateRequest struct {
	AdminContact string `json:"adminContact"`
}

type runtimeSecretRotateRequest struct {
	RotateJWTSecret    bool `json:"rotateJWTSecret"`
	RotatePasswordSalt bool `json:"rotatePasswordSalt"`
}

type sshGatewayUpdateRequest struct {
	ExternalEnabled     bool   `json:"externalEnabled"`
	ExternalPort        string `json:"externalPort"`
	HostFallbackEnabled bool   `json:"hostFallbackEnabled"`
	HostSshdPort        string `json:"hostSshdPort"`
}

// RegisterAdmin attaches admin and manager routes to the versioned API router.
// api is the /api/v1 router group.
// deps provides auth, config, and Kubernetes dependencies.
// This function is used by RegisterV1 for admin dashboard and settings pages.
func RegisterAdmin(api *gin.RouterGroup, deps Dependencies) {
	admin := api.Group("/admin")

	admin.GET("/users", RequireAccessLevel(deps, auth.AccessLevelManager), adminUserListHandler(deps))
	admin.PATCH("/users/:name/access-level", RequireAccessLevel(deps, auth.AccessLevelManager), adminUserAccessLevelHandler(deps))
	admin.DELETE("/users/:name", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminUserDeleteHandler(deps))

	admin.GET("/vms", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminVMListHandler(deps))
	admin.PATCH("/vms/:namespace/:name/power", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminVMPowerHandler(deps))
	admin.DELETE("/vms/:namespace/:name", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminVMDeleteHandler(deps))

	admin.GET("/settings", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminSettingsGetHandler(deps))
	admin.POST("/domain", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminDomainUpdateHandler(deps))
	admin.POST("/admin-contact", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminContactUpdateHandler(deps))
	admin.POST("/https", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminHTTPSUpdateHandler(deps))
	admin.POST("/ssh-gateway", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminSSHGatewayUpdateHandler(deps))
	admin.POST("/runtime-secrets/rotate", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminRuntimeSecretRotateHandler(deps))
	admin.POST("/cert", RequireAccessLevel(deps, auth.AccessLevelAdmin), adminCertUpdateHandler(deps))
}

// adminUserListHandler returns all KiteUser records for the admin dashboard.
// deps provides Kubernetes access through the account service.
// Password hashes are excluded by the account service.
// This handler is used by the user management table.
func adminUserListHandler(deps Dependencies) gin.HandlerFunc {
	return userListHandler(deps)
}

// adminUserAccessLevelHandler updates one user's access level.
// deps provides Kubernetes access through the account service.
// The route parameter may be KiteUser metadata.name or spec.username.
// This handler is used by the admin dashboard access-level select control.
func adminUserAccessLevelHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		var req accessLevelUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}
		if !canCurrentUserChangeAccessLevel(c, deps, c.Param("name"), req.AccessLevel) {
			return
		}

		userName, ok := resolveUserResourceName(c, deps, c.Param("name"))
		if !ok {
			return
		}

		level := req.AccessLevel
		user, err := accountService.Update(c.Request.Context(), userName, account.UpdateRequest{
			AccessLevel: &level,
		})
		if err != nil {
			writeAccountError(c, err, "failed to update user access level")
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

// adminUserDeleteHandler deletes one user and its child VM CRDs.
// deps provides Kubernetes access through the account service.
// The route parameter may be KiteUser metadata.name or spec.username.
// This handler is used by the admin dashboard delete action.
func adminUserDeleteHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		userName, ok := resolveUserResourceName(c, deps, c.Param("name"))
		if !ok {
			return
		}

		if err := accountService.Delete(c.Request.Context(), userName); err != nil {
			writeAccountError(c, err, "failed to delete kite user")
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "kite user deleted successfully"})
	}
}

// adminVMListHandler returns every KiteVirtualMachine in the cluster.
// deps provides Kubernetes access through the VM service.
// This handler is used by the global VM control table.
func adminVMListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		vms, err := vmServiceFromDependencies(deps).ListAll(c.Request.Context())
		if err != nil {
			writeVMError(c, err, "failed to list virtual machines")
			return
		}

		c.JSON(http.StatusOK, gin.H{"vms": vms})
	}
}

// adminVMPowerHandler updates one VM power state by namespace and name.
// deps provides Kubernetes access through the VM service.
// The body must contain powerState set to On or Off.
// This handler is used by manager and admin force stop/start actions.
func adminVMPowerHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req vmPowerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		vm, err := vmServiceFromDependencies(deps).Update(c.Request.Context(), c.Param("namespace"), c.Param("name"), vmservice.UpdateRequest{
			PowerState: &req.PowerState,
		})
		if err != nil {
			writeVMError(c, err, "failed to update virtual machine power state")
			return
		}

		c.JSON(http.StatusOK, gin.H{"vm": vm})
	}
}

// adminVMDeleteHandler marks one VM for controller-managed deletion.
// deps provides Kubernetes access through the VM service.
// namespace and name are route parameters from the global VM table.
// This handler sets spec.delete=true instead of bypassing controller cleanup.
func adminVMDeleteHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		vm, err := vmServiceFromDependencies(deps).MarkDelete(c.Request.Context(), c.Param("namespace"), c.Param("name"))
		if err != nil {
			writeVMError(c, err, "failed to delete virtual machine")
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "virtual machine delete requested", "vm": vm})
	}
}

// configGetHandler returns frontend-readable platform settings.
// deps provides Kubernetes access through the platform settings service.
// This route is public so the frontend can render domain placeholders before login.
// Sensitive TLS material is never returned.
func configGetHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, err := platform.NewService(deps.DynamicClient).GetPublic(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read platform settings"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"config": settings})
	}
}

// adminSettingsGetHandler returns platform settings for the admin settings page.
// deps provides Kubernetes access through the platform settings service.
// Sensitive TLS material is not returned; only whether it exists.
func adminSettingsGetHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, err := platform.NewService(deps.DynamicClient).Get(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read platform settings"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"config": settings})
	}
}

// adminDomainUpdateHandler updates the platform base domain ConfigMap.
// deps provides Kubernetes access through the platform settings service.
// The body contains baseDomain without protocol.
// This handler is used by the admin settings domain form.
func adminDomainUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domainUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		settings, err := platform.NewService(deps.DynamicClient).UpdateBaseDomain(c.Request.Context(), req.BaseDomain)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "base domain updated", "config": settings})
	}
}

// adminContactUpdateHandler updates the operator contact string shown to users without VM create access.
// deps provides Kubernetes access through the platform settings service.
// The body contains adminContact as a free-form contact channel.
// This handler is used by the admin settings contact form.
func adminContactUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminContactUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		settings, err := platform.NewService(deps.DynamicClient).UpdateAdminContact(c.Request.Context(), req.AdminContact)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "admin contact updated", "config": settings})
	}
}

// adminHTTPSUpdateHandler updates the platform HTTPS redirect policy.
// deps provides Kubernetes access through the platform settings service.
// The body contains forceHttps and may explicitly set it to false.
// This handler is used by the admin settings HTTPS enforcement toggle.
func adminHTTPSUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req httpsUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		settings, err := platform.NewService(deps.DynamicClient).UpdateForceHTTPS(c.Request.Context(), req.ForceHTTPS)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "HTTPS enforcement updated", "config": settings})
	}
}

// adminSSHGatewayUpdateHandler stores operator desired SSH gateway exposure settings.
// deps provides Kubernetes access through the platform settings service.
// The controller later reconciles the external Service and gateway Deployment from these values.
// This handler is restricted to Level 3 admins by RegisterAdmin.
func adminSSHGatewayUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sshGatewayUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		settings, err := platform.NewService(deps.DynamicClient).UpdateSSHGateway(c.Request.Context(), platform.SSHGatewayDesired{
			ExternalEnabled:     req.ExternalEnabled,
			ExternalPort:        req.ExternalPort,
			HostFallbackEnabled: req.HostFallbackEnabled,
			HostSshdPort:        req.HostSshdPort,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "SSH gateway settings updated", "config": settings})
	}
}

// adminRuntimeSecretRotateHandler rotates generated runtime secrets in the ConfigMap.
// deps provides Kubernetes access through the platform settings service.
// The new values take effect on the next kite-api process start.
// This handler is used by the admin settings runtime configuration controls.
func adminRuntimeSecretRotateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req runtimeSecretRotateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		settings, err := platform.NewService(deps.DynamicClient).RotateRuntimeSecrets(c.Request.Context(), req.RotateJWTSecret, req.RotatePasswordSalt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "runtime secrets rotated", "config": settings})
	}
}

// adminCertUpdateHandler stores wildcard TLS material in a Kubernetes Secret.
// deps provides Kubernetes access through the platform settings service.
// The body contains PEM certificate and private key strings.
// This handler is used by the admin settings certificate form.
func adminCertUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req certUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		settings, err := platform.NewService(deps.DynamicClient).UpdateTLSCertificate(c.Request.Context(), req.TLSCert, req.TLSKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "TLS certificate updated", "config": settings})
	}
}

// resolveUserResourceName resolves a route user identifier to KiteUser metadata.name.
// c is used to write errors.
// deps provides Kubernetes access through account service.
// identifier may already be metadata.name or may be spec.username from frontend tables.
func resolveUserResourceName(c *gin.Context, deps Dependencies, identifier string) (string, bool) {
	accountService, ok := accountServiceFromDependencies(c, deps)
	if !ok {
		return "", false
	}

	if _, err := accountService.Get(c.Request.Context(), identifier); err == nil {
		return identifier, true
	}

	userObject, found, err := accountService.FindByUsername(c.Request.Context(), identifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read kite users"})
		return "", false
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "kite user was not found"})
		return "", false
	}

	return userObject.GetName(), true
}

// canCurrentUserChangeAccessLevel enforces the split between manager and admin user management.
// c provides the current token claims and writes the exact HTTP rejection response.
// identifier is the target user route parameter and requestedLevel is the desired access level.
// The returned value is true when the caller may continue to update the KiteUser.
func canCurrentUserChangeAccessLevel(c *gin.Context, deps Dependencies, identifier string, requestedLevel int) bool {
	claims, ok := currentClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "access token is required"})
		return false
	}
	if requestedLevel < auth.AccessLevelReadOnly || requestedLevel > auth.AccessLevelAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"message": "access level must be between 0 and 3"})
		return false
	}
	if claims.AccessLevel >= auth.AccessLevelAdmin {
		return true
	}
	if requestedLevel > auth.AccessLevelUser {
		c.JSON(http.StatusForbidden, gin.H{"message": "Level 2 managers can only assign access levels 0 or 1"})
		return false
	}

	accountService, ok := accountServiceFromDependencies(c, deps)
	if !ok {
		return false
	}
	userName, ok := resolveUserResourceName(c, deps, identifier)
	if !ok {
		return false
	}
	target, err := accountService.Get(c.Request.Context(), userName)
	if err != nil {
		writeAccountError(c, err, "failed to read target user")
		return false
	}
	if target.AccessLevel > int64(auth.AccessLevelUser) {
		c.JSON(http.StatusForbidden, gin.H{"message": "Level 2 managers can only change users currently at access level 0 or 1"})
		return false
	}

	return true
}
