package main

import (
	"github.com/gin-gonic/gin"
	"kite/cmd/kite-api/apis"
	"kite/internal/auth"
	"kite/internal/config"
	"log"
	"net/http"

	"k8s.io/client-go/dynamic"
)

func main() {
	cfg := config.Load()

	tokenService, err := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatalf("failed to initialize token service: %v", err)
	}

	dynamicClient, err := getDynamicClient()
	if err != nil {
		log.Fatalf("failed to initialize kubernetes client: %v", err)
	}

	r := newRouter(cfg, tokenService, dynamicClient)

	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("Fail to start Server : %v\n", err)
	}
}

func newRouter(cfg config.Config, tokenService *auth.TokenService, dynamicClient dynamic.Interface) *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
	})

	api := r.Group("/api")
	apis.Register(api, apis.Dependencies{
		Config:        cfg,
		TokenService:  tokenService,
		DynamicClient: dynamicClient,
	})

	return r
}
