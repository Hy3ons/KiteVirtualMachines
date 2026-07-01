package apis

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"kite/internal/account"
	"kite/internal/auth"
	"kite/internal/config"
	"kite/internal/platform"
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

const (
	accessTokenCookieName        = "accessToken"
	httpAccessTokenCookieMaxAge  = 10 * time.Minute
	httpsAccessTokenCookieMaxAge = time.Hour
)

type accessTokenCookiePolicy struct {
	secure bool
	maxAge time.Duration
}

func Register(api *gin.RouterGroup, deps Dependencies) {
	api.POST("/login", loginHandler(deps))
	api.POST("/logout", logoutHandler(deps))
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
	authGroup.POST("/logout", logoutHandler(deps))
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

		cookiePolicy := accessTokenCookiePolicyForRequest(c, deps)
		c.Writer.Header().Add("Set-Cookie", cookiePolicy.accessTokenCookie(accessToken))

		c.JSON(http.StatusOK, loginResponse{
			ExpiresIn: int64(deps.Config.AccessTokenTTL.Seconds()),
			ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
			User:      user,
		})
	}
}

func logoutHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookiePolicy := accessTokenCookiePolicyForRequest(c, deps)
		c.Writer.Header().Add("Set-Cookie", cookiePolicy.clearAccessTokenCookie())
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}

// accessTokenCookiePolicyForRequest creates the cookie policy for one auth request.
// c provides TLS and proxy header information for the current request.
// deps provides live platform settings so forceHttps changes take effect without restarting kite-api.
// The returned policy controls Secure and Max-Age for login and logout Set-Cookie headers.
func accessTokenCookiePolicyForRequest(c *gin.Context, deps Dependencies) accessTokenCookiePolicy {
	secure := requestUsesSecureCookie(c, deps)
	maxAge := httpAccessTokenCookieMaxAge
	if secure {
		maxAge = httpsAccessTokenCookieMaxAge
	}

	return accessTokenCookiePolicy{
		secure: secure,
		maxAge: maxAge,
	}
}

// accessTokenCookie builds the HTTP-only access token cookie value.
// p provides the Secure flag and browser retention time for this request.
// accessToken is the signed Bearer token issued by TokenService.
// The returned string is added to Set-Cookie by loginHandler.
func (p accessTokenCookiePolicy) accessTokenCookie(accessToken string) string {
	return (&http.Cookie{
		Name:     accessTokenCookieName,
		Value:    "Bearer " + accessToken,
		Path:     "/",
		MaxAge:   int(p.maxAge.Seconds()),
		HttpOnly: true,
		Secure:   p.secure,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

// clearAccessTokenCookie builds the Set-Cookie value that removes the access token.
// p provides the Secure flag so logout clears the same browser cookie namespace used by login.
// The returned string is added to Set-Cookie by logoutHandler.
func (p accessTokenCookiePolicy) clearAccessTokenCookie() string {
	return (&http.Cookie{
		Name:     accessTokenCookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   p.secure,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

// requestUsesSecureCookie decides whether auth cookies must include the Secure attribute.
// c provides TLS and proxy header information for the current request.
// deps provides live platform settings so forceHttps changes take effect without restarting kite-api.
// The returned value is used by the access token cookie policy factory.
func requestUsesSecureCookie(c *gin.Context, deps Dependencies) bool {
	if c.Request.TLS != nil {
		return true
	}
	if strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		return true
	}
	if deps.DynamicClient == nil {
		return false
	}
	settings, err := platform.NewService(deps.DynamicClient).Get(c.Request.Context())
	if err != nil {
		return false
	}
	return settings.ForceHTTPS
}
