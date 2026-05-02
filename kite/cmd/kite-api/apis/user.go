package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kite/internal/auth"
	"kite/internal/store"
)

type kiteUserResponse struct {
	Name         string `json:"name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Namespace    string `json:"namespace"`
	ProfileImage string `json:"profile_image"`
	AccessLevel  int64  `json:"access_level"`
}

// RegisterUsers attaches KiteUser HTTP routes to the API router group.
// api is the /api router group created during kite-api startup.
// deps provides authentication, authorization, configuration, and Kubernetes clients for the handlers.
// This function is used by Register when the API server starts.
func RegisterUsers(api *gin.RouterGroup, deps Dependencies) {
	api.GET("/users", RequireAccessLevel(deps, auth.AccessLevelManager), userListHandler(deps))
	api.POST("/users", RequireAccessLevel(deps, auth.AccessLevelAdmin), userSignUpHandler(deps))
	api.POST("/user", userSignUpHandler(deps))
}

type kiteUserSignUpRequest struct {
	Name      string `json:"name" binding:"required"`
	Username  string `json:"username" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
}

// userSignUpHandler creates a KiteUser CRD from a signup request.
// deps provides the dynamic Kubernetes client and password salt from API startup.
// The handler writes only the KiteUser CRD; namespace and policy resources are reconciled by kite-controller.
func userSignUpHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.DynamicClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"message": "kubernetes client is not configured",
			})
			return
		}

		if deps.Config.PasswordSalt == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "password salt is not configured",
			})
			return
		}

		var req kiteUserSignUpRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid request body",
			})
			return
		}

		userStore := store.NewUserStore(deps.DynamicClient)

		created, err := userStore.Create(c.Request.Context(), store.KiteUserRecord{
			Name: req.Name,
			Spec: store.KiteUserSpec{
				Username:     req.Username,
				Email:        req.Email,
				Password:     auth.HashPassword(req.Password, deps.Config.PasswordSalt),
				Namespace:    req.Namespace,
				ProfileImage: "",
				AccessLevel:  1, // 기본유저
			},
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to create kite user",
			})
			return
		}

		spec, ok := created.Object["spec"].(map[string]any)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "failed to parse kite user spec",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "kite user created successfully",
			"user": gin.H{
				"name":          created.GetName(),
				"username":      stringValue(spec, "username"),
				"email":         stringValue(spec, "email"),
				"namespace":     stringValue(spec, "namespace"),
				"profile_image": stringValue(spec, "profile_image"),
				"access_level":  intValue(spec, "access_level"),
			},
		})
	}
}

// userListHandler lists KiteUser CRDs for manager-level API callers.
// deps provides the dynamic Kubernetes client used to read cluster-scoped KiteUser resources.
// The response intentionally omits password hashes from each user.
func userListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.DynamicClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"message": "kubernetes client is not configured",
			})
			return
		}

		userStore := store.NewUserStore(deps.DynamicClient)
		list, err := userStore.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to list kite users",
			})
			return
		}

		users := make([]kiteUserResponse, 0, len(list.Items))
		for _, item := range list.Items {
			spec, ok := item.Object["spec"].(map[string]any)

			if !ok {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"message": "failed to parse kite user spec",
				})
				return
			}

			users = append(users, kiteUserResponse{
				Name:         item.GetName(),
				Username:     stringValue(spec, "username"),
				Email:        stringValue(spec, "email"),
				Namespace:    stringValue(spec, "namespace"),
				ProfileImage: stringValue(spec, "profile_image"),
				AccessLevel:  intValue(spec, "access_level"),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"users": users,
		})
	}
}

// stringValue reads a string field from unstructured CRD data.
// data is a map from a Kubernetes object field such as spec.
// key is the field name to read.
// The returned value is empty when the field is missing or not a string.
func stringValue(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

// intValue reads an integer-like field from unstructured CRD data.
// data is a map from a Kubernetes object field such as spec.
// key is the field name to read.
// The returned value is zero when the field is missing or not numeric.
func intValue(data map[string]any, key string) int64 {
	switch value := data[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}
