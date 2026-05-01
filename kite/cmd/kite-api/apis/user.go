package apis

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kite/internal/auth"
)

type kiteUserResponse struct {
	Name         string `json:"name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Namespace    string `json:"namespace"`
	ProfileImage string `json:"profile_image"`
	AccessLevel  int64  `json:"access_level"`
}

var kiteUserGVR = schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kiteusers",
}

func RegisterUsers(api *gin.RouterGroup, deps Dependencies) {
	users := api.Group("/users")
	users.Use(RequireAccessLevel(deps, auth.AccessLevelManager))
	users.GET("", userListHandler(deps))
}

func userListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.DynamicClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"message": "kubernetes client is not configured",
			})
			return
		}

		list, err := deps.DynamicClient.Resource(kiteUserGVR).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to list kite users",
			})
			return
		}

		users := make([]kiteUserResponse, 0, len(list.Items))
		for _, item := range list.Items {
			spec, _ := item.Object["spec"].(map[string]any)
			users = append(users, kiteUserResponse{
				Name:         item.GetName(),
				Username:     stringValue(spec, "username"),
				Email:        stringValue(spec, "email"),
				Password:     stringValue(spec, "password"),
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

func stringValue(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

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
