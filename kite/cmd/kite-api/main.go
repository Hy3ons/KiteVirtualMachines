package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"kite/cmd/kite-api/apis"
	"kite/internal/auth"
	"kite/internal/config"
	"kite/internal/health"
)

const defaultAPIListenAddress = ":8080"

func main() {
	restConfig, err := getClusterConfig()
	if err != nil {
		log.Fatalf("failed to load kubernetes config: %v", err)
	}

	dynamicClient, err := getDynamicClient(restConfig)
	if err != nil {
		log.Fatalf("failed to initialize kubernetes client: %v", err)
	}

	cfg, err := config.Bootstrap(context.Background(), dynamicClient)
	if err != nil {
		log.Fatalf("failed to initialize runtime config: %v", err)
	}

	tokenService, err := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatalf("failed to initialize token service: %v", err)
	}

	r := newRouter(cfg, tokenService, dynamicClient, restConfig)

	if err := r.Run(defaultAPIListenAddress); err != nil {
		log.Fatalf("Fail to start Server : %v\n", err)
	}
}

func newRouter(cfg config.Config, tokenService *auth.TokenService, dynamicClient dynamic.Interface, restConfig *rest.Config) *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.Use(corsMiddleware())
	r.Use(csrfOriginMiddleware())

	r.GET("/health", func(c *gin.Context) {
		report := health.Run(c.Request.Context(), dynamicClient)
		status := http.StatusOK
		if report.Status != "ok" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, report)
	})
	r.GET("/api/v1/health", func(c *gin.Context) {
		report := health.Run(c.Request.Context(), dynamicClient)
		status := http.StatusOK
		if report.Status != "ok" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, report)
	})

	api := r.Group("/api")
	apis.Register(api, apis.Dependencies{
		Config:        cfg,
		TokenService:  tokenService,
		DynamicClient: dynamicClient,
		RestConfig:    restConfig,
	})

	return r
}

// csrfOriginMiddleware rejects unsafe browser requests from another origin.
// The middleware allows local Vite development origins and exact same-origin platform requests.
// Requests without an Origin header are allowed so CLI and server-side clients keep working.
func csrfOriginMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !methodCanMutateState(c.Request.Method) {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" || isAllowedOrigin(origin) || originMatchesRequestHost(origin, c.Request.Host) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "cross-origin state-changing request is not allowed"})
	}
}

// methodCanMutateState reports whether a HTTP method should be protected by Origin checks.
// method is the request method from net/http.
// The returned value is false for read-only and CORS preflight methods.
func methodCanMutateState(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// originMatchesRequestHost reports whether Origin matches the request Host exactly.
// origin is parsed as a browser Origin header.
// host is the Host header that reached kite-api through ingress or port-forwarding.
func originMatchesRequestHost(origin string, host string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, strings.TrimSpace(host))
}

// corsMiddleware handles browser CORS preflight requests for the Vite frontend.
// The middleware allows local development origins and same-origin production calls.
// OPTIONS requests return 204 before route matching so POST/PATCH/DELETE endpoints do not log 404.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isAllowedOrigin reports whether an Origin header is accepted by the API server.
// origin is the browser-provided origin value.
// The returned bool is used by corsMiddleware before writing CORS response headers.
func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	return origin == "http://localhost:5173" ||
		origin == "http://127.0.0.1:5173" ||
		strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:")
}
