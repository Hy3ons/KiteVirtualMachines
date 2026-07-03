package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"kite/internal/auth"
	"kite/internal/offer"
	vmservice "kite/internal/vm"
)

type vmOfferCreateRequest struct {
	TargetUser      string `json:"targetUser"`
	TargetNamespace string `json:"targetNamespace"`
	CPU             int    `json:"cpu" binding:"required"`
	Memory          string `json:"memory" binding:"required"`
	Disk            string `json:"disk" binding:"required"`
	Image           string `json:"image"`
	ExpiresAt       string `json:"expiresAt"`
}

type vmOfferClaimRequest struct {
	VMName               string `json:"vmName" binding:"required"`
	DomainPrefix         string `json:"domainPrefix"`
	SSHID                string `json:"sshId" binding:"required"`
	InitialLoginPassword string `json:"initialLoginPassword" binding:"required"`
	PowerState           string `json:"powerState"`
}

// RegisterVirtualMachineOffers attaches user and admin VM offer routes to the versioned API router.
// api is the /api/v1 router group.
// deps provides auth and Kubernetes dependencies.
// This function is used by RegisterV1 for admin assigned VM capacity offers.
func RegisterVirtualMachineOffers(api *gin.RouterGroup, deps Dependencies) {
	api.GET("/vm-offers", RequireAccessLevel(deps, auth.AccessLevelReadOnly), vmOfferListHandler(deps))
	api.POST("/vm-offers/:name/claim", RequireAccessLevel(deps, auth.AccessLevelReadOnly), vmOfferClaimHandler(deps))

	admin := api.Group("/admin/vm-offers", RequireAccessLevel(deps, auth.AccessLevelAdmin))
	admin.POST("", vmOfferCreateHandler(deps))
	admin.DELETE("/:namespace/:name", vmOfferDeleteHandler(deps))
}

// vmOfferListHandler returns offers assigned to the authenticated user's namespace.
// deps provides Kubernetes access through the offer service.
// The namespace is loaded from the current KiteUser rather than request input.
// This handler is used by the user dashboard offer section.
func vmOfferListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		offers, err := offerServiceFromDependencies(deps).List(c.Request.Context(), user.Namespace)
		if err != nil {
			writeOfferError(c, err, "failed to list VM offers")
			return
		}

		c.JSON(http.StatusOK, gin.H{"offers": offers})
	}
}

// vmOfferClaimHandler claims one offer and creates the final KiteVirtualMachine.
// deps provides Kubernetes access through the offer and VM services.
// The route parameter is metadata.name in the authenticated user's namespace.
// This handler is used by the user dashboard claim modal.
func vmOfferClaimHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		var req vmOfferClaimRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		vm, err := offerServiceFromDependencies(deps).Claim(c.Request.Context(), user.Namespace, c.Param("name"), user.Username, offer.ClaimRequest{
			VMName:               req.VMName,
			DomainPrefix:         req.DomainPrefix,
			SSHID:                req.SSHID,
			InitialLoginPassword: req.InitialLoginPassword,
			PowerState:           req.PowerState,
		})
		if err != nil {
			writeOfferError(c, err, "failed to claim VM offer")
			return
		}

		c.JSON(http.StatusCreated, gin.H{"vm": vm})
	}
}

// vmOfferCreateHandler creates an offer in a target user's namespace.
// deps provides Kubernetes access through account and offer services.
// The request may identify the target by username or namespace.
// This handler is used by the level 3 admin dashboard offer form.
func vmOfferCreateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req vmOfferCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		targetNamespace, ok := targetOfferNamespace(c, deps, req)
		if !ok {
			return
		}
		claims, ok := currentClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "access token is required"})
			return
		}

		created, err := offerServiceFromDependencies(deps).Create(c.Request.Context(), offer.CreateRequest{
			TargetNamespace: targetNamespace,
			CPU:             req.CPU,
			Memory:          req.Memory,
			Disk:            req.Disk,
			Image:           req.Image,
			ExpiresAt:       req.ExpiresAt,
			CreatedBy:       claims.Subject,
		})
		if err != nil {
			writeOfferError(c, err, "failed to create VM offer")
			return
		}

		c.JSON(http.StatusCreated, gin.H{"offer": created})
	}
}

// vmOfferDeleteHandler deletes one offer from a target namespace.
// deps provides Kubernetes access through the offer service.
// namespace and name are route parameters selected from the admin table.
// This handler is restricted to level 3 admins.
func vmOfferDeleteHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := offerServiceFromDependencies(deps).Delete(c.Request.Context(), c.Param("namespace"), c.Param("name")); err != nil {
			writeOfferError(c, err, "failed to delete VM offer")
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "VM offer deleted"})
	}
}

// offerServiceFromDependencies creates an offer service for one request.
// deps provides the dynamic Kubernetes client and password salt.
// The returned service reads offer CRDs and creates KiteVirtualMachine CRDs during claim.
func offerServiceFromDependencies(deps Dependencies) *offer.Service {
	return offer.NewService(deps.DynamicClient, deps.Config.PasswordSalt)
}

// targetOfferNamespace resolves the admin create request to one KiteUser namespace.
// c is used to write validation errors.
// req may provide TargetUser or TargetNamespace.
// The returned namespace is where the offer CRD should be created.
func targetOfferNamespace(c *gin.Context, deps Dependencies, req vmOfferCreateRequest) (string, bool) {
	if req.TargetNamespace != "" {
		return req.TargetNamespace, true
	}
	if req.TargetUser == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "targetUser or targetNamespace is required"})
		return "", false
	}

	userName, ok := resolveUserResourceName(c, deps, req.TargetUser)
	if !ok {
		return "", false
	}
	accountService, ok := accountServiceFromDependencies(c, deps)
	if !ok {
		return "", false
	}
	user, err := accountService.Get(c.Request.Context(), userName)
	if err != nil {
		writeAccountError(c, err, "failed to read target user")
		return "", false
	}
	return user.Namespace, true
}

// writeOfferError maps offer, VM, and Kubernetes errors to HTTP responses.
// c is the active Gin request context.
// err is returned by internal/offer, internal/vm, or Kubernetes store code.
// fallbackMessage is used for unexpected internal errors.
func writeOfferError(c *gin.Context, err error, fallbackMessage string) {
	if apierrors.IsNotFound(err) || vmservice.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"message": "VM offer was not found"})
		return
	}
	if kind, ok := offer.RequestErrorKind(err); ok {
		switch kind {
		case offer.ErrorKindInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		case offer.ErrorKindConflict:
			c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": fallbackMessage})
		}
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
