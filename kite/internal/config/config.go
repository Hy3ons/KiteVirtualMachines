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
	PasswordSalt   string
	AdminUsername  string
	AdminPassword  string
	AdminAccess    int
	AccessTokenTTL time.Duration
}

// Load reads kite-api configuration from environment variables.
// KITE_HTTP_ADDR controls the HTTP bind address.
// KITE_JWT_SECRET signs API access tokens.
// KITE_PASSWORD_SALT is used when hashing KiteUser passwords.
// The returned Config is used by API server startup and handlers.
func Load() Config {
	return Config{
		HTTPAddr:       getEnv("KITE_HTTP_ADDR", DefaultHTTPAddr),
		JWTSecret:      os.Getenv("KITE_JWT_SECRET"),
		PasswordSalt:   os.Getenv("KITE_PASSWORD_SALT"),
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
