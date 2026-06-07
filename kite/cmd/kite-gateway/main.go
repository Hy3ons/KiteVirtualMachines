package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"kite/internal/config"
	"kite/internal/gateway"
	"kite/internal/kube"
)

const (
	defaultListenAddress        = ":2222"
	defaultBackendTimeoutSecond = 90
	defaultBackendRetrySecond   = 2
	defaultHostFallbackTimeout  = 5
)

// main starts kite-gateway.
// The process creates Kubernetes clients, starts the KiteVM route informer, and serves SSH traffic.
// It exits when SIGINT or SIGTERM is received.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	clientManager, err := kube.GetClientManager()
	if err != nil {
		log.Fatalf("failed to create Kubernetes clients: %v", err)
	}
	cfg, err := config.Bootstrap(ctx, clientManager.DynamicClient)
	if err != nil {
		log.Fatalf("failed to load runtime config: %v", err)
	}

	routeTable := gateway.NewRouteTable(cfg.PasswordSalt)
	go func() {
		if err := gateway.RunRouteInformer(ctx, clientManager.DynamicClient, routeTable); err != nil && ctx.Err() == nil {
			log.Fatalf("failed to run KiteVM route informer: %v", err)
		}
	}()

	server, err := gateway.NewServer(gateway.ServerConfig{
		ListenAddress:        envString("KITE_GATEWAY_LISTEN_ADDRESS", defaultListenAddress),
		HostKeyPath:          envString("KITE_GATEWAY_HOST_KEY_PATH", ""),
		BackendTimeout:       time.Duration(envInt("KITE_GATEWAY_BACKEND_TIMEOUT_SECONDS", defaultBackendTimeoutSecond)) * time.Second,
		BackendRetryInterval: time.Duration(envInt("KITE_GATEWAY_BACKEND_RETRY_SECONDS", defaultBackendRetrySecond)) * time.Second,
		HostFallbackEnabled:  envBool("KITE_GATEWAY_HOST_FALLBACK_ENABLED", true),
		HostFallbackAddress:  envString("KITE_GATEWAY_HOST_SSHD_ADDRESS", ""),
		HostFallbackTimeout:  time.Duration(envInt("KITE_GATEWAY_HOST_FALLBACK_TIMEOUT_SECONDS", defaultHostFallbackTimeout)) * time.Second,
	}, clientManager.DynamicClient, routeTable)
	if err != nil {
		log.Fatalf("failed to initialize gateway server: %v", err)
	}

	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatalf("kite-gateway stopped with error: %v", err)
	}
}

// envString reads one string environment variable.
// name is the environment variable key.
// fallback is returned when the variable is empty.
// This helper is used by kite-gateway startup configuration.
func envString(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

// envInt reads one positive integer environment variable.
// name is the environment variable key.
// fallback is returned when the variable is empty, invalid, or not positive.
// This helper is used for timeout configuration in kite-gateway startup.
func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// envBool reads one boolean environment variable.
// name is the environment variable key.
// fallback is returned when the variable is empty or not a recognized boolean value.
// This helper is used for optional gateway features such as host sshd fallback.
func envBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	switch value {
	case "true", "1", "yes", "YES", "y", "Y":
		return true
	case "false", "0", "no", "NO", "n", "N":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}
