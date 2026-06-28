package apis

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"kite/internal/account"
	"kite/internal/auth"
	"kite/internal/config"
)

type Dependencies struct {
	Config           config.Config
	TokenService     *auth.TokenService
	DynamicClient    dynamic.Interface
	RestConfig       *rest.Config
	ConsoleTickets   *ConsoleTicketService
	ConsoleConnector ConsoleConnector
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	ExpiresIn int64  `json:"expiresIn"`
	ExpiresAt string `json:"expiresAt"`
	User      any    `json:"user,omitempty"`
}

func Register(api *gin.RouterGroup, deps Dependencies) {
	api.POST("/login", loginHandler(deps))
	api.POST("/logout", logoutHandler())
	RegisterUsers(api, deps)

	v1 := api.Group("/v1")
	RegisterV1(v1, deps)
}

// RegisterV1 attaches the frontend-facing versioned API routes.
// api is the /api/v1 router group created during kite-api startup.
// deps provides shared Kubernetes, config, and auth dependencies.
// This function is used by Register while legacy /api routes remain available.
func RegisterV1(api *gin.RouterGroup, deps Dependencies) {
	deps = withConsoleDefaults(deps)
	api.GET("/config", configGetHandler(deps))
	api.GET("/me", RequireAccessLevel(deps, auth.AccessLevelReadOnly), currentUserHandler(deps))

	authGroup := api.Group("/auth")
	authGroup.POST("/login", loginHandler(deps))
	authGroup.POST("/logout", logoutHandler())
	authGroup.POST("/signup", userSignUpHandler(deps))

	RegisterVirtualMachines(api, deps)
	RegisterAdmin(api, deps)
}

// withConsoleDefaults fills optional console dependencies for production startup.
// deps carries shared API dependencies from startup or tests.
// The returned Dependencies has a signed ticket service and connector when enough Kubernetes config exists.
// This function is used before route registration so handlers can stay small and deterministic.
func withConsoleDefaults(deps Dependencies) Dependencies {
	if deps.ConsoleTickets == nil {
		deps.ConsoleTickets = NewConsoleTicketService(30*time.Second, deps.Config.JWTSecret)
	}
	if deps.ConsoleConnector == nil && deps.RestConfig != nil {
		deps.ConsoleConnector = NewKubeVirtConsoleConnector(deps.RestConfig)
	}
	return deps
}

// loginHandler authenticates a KiteUser and issues an API access token.
// deps provides the dynamic Kubernetes client, password salt, and token service from API startup.
// The request email is matched against KiteUser spec.email and the password is verified against spec.password.
// This handler is used by the frontend login page and stores the access token in an HTTP-only cookie.
func loginHandler(deps Dependencies) gin.HandlerFunc {
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

		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "email and password are required",
			})
			return
		}

		accountService := account.NewService(deps.DynamicClient, deps.Config.PasswordSalt)
		user, ok, err := accountService.Authenticate(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to read kite users",
			})
			return
		}
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "invalid email or password",
			})
			return
		}

		accessToken, expiresAt, err := deps.TokenService.IssueAccessToken(user.Username, int(user.AccessLevel))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to issue access token",
			})
			return
		}

		c.Writer.Header().Add("Set-Cookie", accessTokenCookie(accessToken, int(deps.Config.AccessTokenTTL.Seconds()), requestUsesSecureCookie(c)))

		c.JSON(http.StatusOK, loginResponse{
			ExpiresIn: int64(deps.Config.AccessTokenTTL.Seconds()),
			ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
			User:      user,
		})
	}
}

func logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Add("Set-Cookie", clearAccessTokenCookie(requestUsesSecureCookie(c)))
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}

// accessTokenCookie builds the HTTP-only access token cookie value.
// accessToken is the signed Bearer token issued by TokenService.
// maxAge is the cookie lifetime in seconds.
// secure controls whether the Secure attribute is emitted for HTTPS requests.
// The returned string is added to Set-Cookie by loginHandler.
func accessTokenCookie(accessToken string, maxAge int, secure bool) string {
	return "accessToken=\"Bearer " + accessToken + "\"; Path=/; Max-Age=" + strconv.Itoa(maxAge) + "; HttpOnly" + secureCookieAttribute(secure) + "; SameSite=Lax"
}

func clearAccessTokenCookie(secure bool) string {
	return "accessToken=; Path=/; Max-Age=0; HttpOnly" + secureCookieAttribute(secure) + "; SameSite=Lax"
}

func secureCookieAttribute(secure bool) string {
	if secure {
		return "; Secure"
	}
	return ""
}

func requestUsesSecureCookie(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}
