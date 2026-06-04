package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"kite/internal/account"
	"kite/internal/auth"
)

type kiteUserResponse = account.PublicUser

type kiteUserSignUpRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type kiteUserUpdateRequest struct {
	Email        *string `json:"email"`
	Password     *string `json:"password"`
	Namespace    *string `json:"namespace"`
	ProfileImage *string `json:"profile_image"`
	AccessLevel  *int    `json:"access_level"`
}

// RegisterUsers attaches KiteUser HTTP routes to the API router group.
// api is the /api router group created during kite-api startup.
// deps provides authentication, authorization, configuration, and Kubernetes clients for the handlers.
// This function is used by Register when the API server starts.
func RegisterUsers(api *gin.RouterGroup, deps Dependencies) {
	api.GET("/users", RequireAccessLevel(deps, auth.AccessLevelManager), userListHandler(deps))
	api.GET("/users/:name", RequireAccessLevel(deps, auth.AccessLevelReadOnly), userGetHandler(deps))
	api.POST("/users", RequireAccessLevel(deps, auth.AccessLevelAdmin), userSignUpHandler(deps))
	api.PATCH("/users/:name", RequireAccessLevel(deps, auth.AccessLevelAdmin), userUpdateHandler(deps))
	api.DELETE("/users/:name", RequireAccessLevel(deps, auth.AccessLevelAdmin), userDeleteHandler(deps))
	api.GET("/me", RequireAccessLevel(deps, auth.AccessLevelReadOnly), currentUserHandler(deps))
	api.POST("/signup", userSignUpHandler(deps))
	api.POST("/user", userSignUpHandler(deps))
}

// userSignUpHandler creates a KiteUser CRD from a signup request.
// deps provides the dynamic Kubernetes client and password salt from API startup.
// The handler delegates CRD creation and first-user access-level decisions to internal/account.
// Namespace and policy resources are reconciled later by kite-controller.
func userSignUpHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		var req kiteUserSignUpRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid request body",
			})
			return
		}

		user, err := accountService.SignUp(c.Request.Context(), account.SignUpRequest{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			writeAccountError(c, err, "failed to create kite user")
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "kite user created successfully",
			"user":    user,
		})
	}
}

// userListHandler lists KiteUser CRDs for manager-level API callers.
// deps provides the dynamic Kubernetes client used by internal/account.
// The response intentionally omits password hashes from each user.
// This handler is used by future administrator and manager user list pages.
func userListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		users, err := accountService.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to list kite users",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"users": users,
		})
	}
}

// userGetHandler returns one KiteUser for an authorized caller.
// deps provides the dynamic Kubernetes client used by internal/account.
// Managers and admins may read any user; lower access levels may only read their own username.
// This handler is used by frontend profile and admin user detail pages.
func userGetHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		user, err := accountService.Get(c.Request.Context(), c.Param("name"))
		if err != nil {
			writeAccountError(c, err, "failed to read kite user")
			return
		}

		if !canReadUser(c, user) {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "cannot read another user",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	}
}

// currentUserHandler returns the KiteUser that matches the current token subject.
// deps provides the dynamic Kubernetes client used by internal/account.
// The response omits the stored password hash.
// This handler is used by frontend startup code to load the signed-in user profile.
func currentUserHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := currentClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "access token is required",
			})
			return
		}

		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		userObject, found, err := accountService.FindByUsername(c.Request.Context(), claims.Subject)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to read kite users",
			})
			return
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "current user was not found",
			})
			return
		}

		user, err := accountService.Get(c.Request.Context(), userObject.GetName())
		if err != nil {
			writeAccountError(c, err, "failed to read kite user")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	}
}

// userUpdateHandler updates mutable KiteUser fields for an admin caller.
// deps provides the dynamic Kubernetes client and password salt used by internal/account.
// The route parameter selects metadata.name of the cluster-scoped KiteUser resource.
// This handler is intended for the future administrator user management page.
func userUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		var req kiteUserUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid request body",
			})
			return
		}

		user, err := accountService.Update(c.Request.Context(), c.Param("name"), account.UpdateRequest{
			Email:        req.Email,
			Password:     req.Password,
			Namespace:    req.Namespace,
			ProfileImage: req.ProfileImage,
			AccessLevel:  req.AccessLevel,
		})
		if err != nil {
			writeAccountError(c, err, "failed to update kite user")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	}
}

// userDeleteHandler deletes one KiteUser CRD and its child VM CRDs for an admin caller.
// deps provides the dynamic Kubernetes client used by internal/account.
// The route parameter selects metadata.name of the KiteUser resource.
// internal/account deletes KiteVirtualMachine CRDs in the user's namespace before deleting the user.
func userDeleteHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountService, ok := accountServiceFromDependencies(c, deps)
		if !ok {
			return
		}

		if err := accountService.Delete(c.Request.Context(), c.Param("name")); err != nil {
			writeAccountError(c, err, "failed to delete kite user")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "kite user deleted successfully",
		})
	}
}

// accountServiceFromDependencies creates an account service for a request.
// c is used to write an HTTP error response when Kubernetes access is unavailable.
// deps contains the dynamic Kubernetes client and password salt provided by API startup.
// The returned boolean is false when the caller should stop handling the request.
func accountServiceFromDependencies(c *gin.Context, deps Dependencies) (*account.Service, bool) {
	if deps.DynamicClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"message": "kubernetes client is not configured",
		})
		return nil, false
	}

	return account.NewService(deps.DynamicClient, deps.Config.PasswordSalt), true
}

// canReadUser checks whether the current token may read a user response.
// c provides the token claims set by RequireAccessLevel.
// user is the response object that would be returned to the client.
// Managers and admins can read all users; lower levels can only read their own username.
func canReadUser(c *gin.Context, user kiteUserResponse) bool {
	claims, ok := currentClaims(c)
	if !ok {
		return false
	}

	return claims.AccessLevel >= auth.AccessLevelManager || claims.Subject == user.Username
}

// writeAccountError maps account service errors to HTTP responses.
// c is the active Gin request context.
// err is returned by internal/account or Kubernetes store code.
// fallbackMessage is used for unexpected internal errors.
func writeAccountError(c *gin.Context, err error, fallbackMessage string) {
	if apierrors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "kite user was not found",
		})
		return
	}

	if kind, ok := account.RequestErrorKind(err); ok {
		switch kind {
		case account.ErrorKindInvalid:
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
		case account.ErrorKindConflict:
			c.JSON(http.StatusConflict, gin.H{
				"message": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": fallbackMessage,
			})
		}
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"message": fallbackMessage,
	})
}
