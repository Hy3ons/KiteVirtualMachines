package apis

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kite/internal/auth"
)

const claimsContextKey = "authClaims"

// RequireAccessLevel creates a gin.HandlerFunc middleware that checks user authorization.
// deps is used to access the token verification service.
// minimumAccessLevel specifies the lowest access level allowed (e.g., auth.AccessLevelManager).
// This function is used to protect API routes that require specific privileges.
func RequireAccessLevel(deps Dependencies, minimumAccessLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)

		// 토큰 존재해야함.
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "access token is required",
			})
			return
		}


		
		claims, err := deps.TokenService.VerifyAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "invalid access token",
			})
			return
		}

		if claims.AccessLevel < minimumAccessLevel {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"message": "manager access level is required",
			})
			return
		}

		c.Set(claimsContextKey, claims)
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return header
	}

	cookie, err := c.Cookie("accessToken")
	if err != nil {
		return ""
	}

	return cookie
}

func currentClaims(c *gin.Context) (auth.Claims, bool) {
	value, ok := c.Get(claimsContextKey)
	if !ok {
		return auth.Claims{}, false
	}

	claims, ok := value.(auth.Claims)
	return claims, ok
}
