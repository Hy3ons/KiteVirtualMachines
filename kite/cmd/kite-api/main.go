package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"

	"kite/cmd/kite-api/apis"
	"kite/internal/auth"
	"kite/internal/config"
	"kite/internal/health"
)

const defaultAPIListenAddress = ":8080"

func main() {
	dynamicClient, err := getDynamicClient()
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

	r := newRouter(cfg, tokenService, dynamicClient)

	if err := r.Run(defaultAPIListenAddress); err != nil {
		log.Fatalf("Fail to start Server : %v\n", err)
	}
}

func newRouter(cfg config.Config, tokenService *auth.TokenService, dynamicClient dynamic.Interface) *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.Use(corsMiddleware())

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
	})

	return r
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
