package apis

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"

	"kite/internal/auth"
	"kite/internal/config"
)

type Dependencies struct {
	Config        config.Config
	TokenService  *auth.TokenService
	DynamicClient dynamic.Interface
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int64  `json:"expiresIn"`
	ExpiresAt   string `json:"expiresAt"`
}

func Register(api *gin.RouterGroup, deps Dependencies) {
	api.POST("/login", loginHandler(deps))
	RegisterUsers(api, deps)
}

func loginHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "username and password are required",
			})
			return
		}

		if !auth.ValidateCredentials(req.Username, req.Password, deps.Config.AdminUsername, deps.Config.AdminPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "invalid username or password",
			})
			return
		}

		accessToken, expiresAt, err := deps.TokenService.IssueAccessToken(req.Username, deps.Config.AdminAccess)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to issue access token",
			})
			return
		}

		c.Writer.Header().Add("Set-Cookie", accessTokenCookie(accessToken, int(deps.Config.AccessTokenTTL.Seconds())))

		c.JSON(http.StatusOK, loginResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   int64(deps.Config.AccessTokenTTL.Seconds()),
			ExpiresAt:   expiresAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

func accessTokenCookie(accessToken string, maxAge int) string {
	return "accessToken=\"Bearer " + accessToken + "\"; Path=/; Max-Age=" + strconv.Itoa(maxAge) + "; HttpOnly; SameSite=Lax"
}
