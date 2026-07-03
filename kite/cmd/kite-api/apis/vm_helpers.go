package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"kite/internal/account"
	"kite/internal/auth"
	vmservice "kite/internal/vm"
)

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

// canMutateOwnVM reports whether an access level may change VMs in its own namespace.
// accessLevel is read from the current KiteUser object rather than trusting only token claims.
// The result is used by VM create, update, delete, and power handlers.
func canMutateOwnVM(accessLevel int64) bool {
	return accessLevel >= int64(auth.AccessLevelUser)
}

// activeVMCount counts VMs that still represent active user allocations.
// vms are already scoped to one user's namespace.
// Deleting VMs are skipped so a pending cleanup does not permanently consume Level 1 quota.
func activeVMCount(vms []vmservice.VirtualMachine) int {
	count := 0
	for _, vm := range vms {
		if !vm.Delete {
			count++
		}
	}
	return count
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
