package config

import (
	"os"
	"time"
)

const (
	DefaultHTTPAddr              = ":8080"
	DefaultAccessTokenTTLMinutes = 60
)

type Config struct {
	HTTPAddr       string
	JWTSecret      string
	AdminUsername  string
	AdminPassword  string
	AdminAccess    int
	AccessTokenTTL time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:       getEnv("KITE_HTTP_ADDR", DefaultHTTPAddr),
		JWTSecret:      os.Getenv("KITE_JWT_SECRET"),
		AdminUsername:  os.Getenv("KITE_ADMIN_USERNAME"),
		AdminPassword:  os.Getenv("KITE_ADMIN_PASSWORD"),
		AdminAccess:    3,
		AccessTokenTTL: time.Minute * DefaultAccessTokenTTLMinutes,
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
