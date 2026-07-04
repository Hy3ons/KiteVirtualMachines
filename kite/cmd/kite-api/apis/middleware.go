package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kite/internal/account"
	"kite/internal/auth"
)

const claimsContextKey = "authClaims"

// RequireAccessLevel creates a gin.HandlerFunc middleware that checks user authorization.
// deps is used to access the token verification service.
// minimumAccessLevel specifies the lowest access level allowed (e.g., auth.AccessLevelManager).
// This function is used to protect API routes that require specific privileges.
func RequireAccessLevel(deps Dependencies, minimumAccessLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := accessTokenCookieValue(c)

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

		claims, ok := refreshClaimsFromCurrentUser(c, deps, claims)
		if !ok {
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

// refreshClaimsFromCurrentUser reloads the token subject from KiteUser CRDs.
// c is the active Gin request context used to write authorization failures.
// deps provides Kubernetes access to read the current KiteUser state.
// claims comes from the signed access token and contributes only the subject identity.
// The returned claims copy uses the current CRD access level so demotion takes effect before token expiry.
func refreshClaimsFromCurrentUser(c *gin.Context, deps Dependencies, claims auth.Claims) (auth.Claims, bool) {
	accountService, ok := accountServiceFromMiddleware(c, deps)
	if !ok {
		return auth.Claims{}, false
	}

	userObject, found, err := accountService.FindByUsername(c.Request.Context(), claims.Subject)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to read kite users"})
		return auth.Claims{}, false
	}
	if !found {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "current user was not found"})
		return auth.Claims{}, false
	}

	user, err := accountService.Get(c.Request.Context(), userObject.GetName())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to read current user"})
		return auth.Claims{}, false
	}

	claims.AccessLevel = int(user.AccessLevel)
	return claims, true
}

// accountServiceFromMiddleware creates an account service for auth middleware.
// c is used to abort the request when Kubernetes access is unavailable.
// deps contains the dynamic Kubernetes client and password salt used by account operations.
// The returned boolean is false when the middleware has already written the response.
func accountServiceFromMiddleware(c *gin.Context, deps Dependencies) (*account.Service, bool) {
	if deps.DynamicClient == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"message": "kubernetes client is not configured",
		})
		return nil, false
	}

	return account.NewService(deps.DynamicClient, deps.Config.PasswordSalt), true
}

func accessTokenCookieValue(c *gin.Context) string {
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
