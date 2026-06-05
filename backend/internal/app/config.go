package app

import (
	"os"
	"time"
)

const (
	defaultAppEnv         = "dev"
	defaultHTTPAddr       = ":8080"
	defaultJWTSecret      = "change-me"
	defaultAccessTokenTTL = 24 * time.Hour
)

type Config struct {
	AppEnv         string
	HTTPAddr       string
	JWTSecret      string
	AccessTokenTTL time.Duration
}

func LoadConfig() Config {
	return Config{
		AppEnv:         getenv("APP_ENV", defaultAppEnv),
		HTTPAddr:       getenv("HTTP_ADDR", defaultHTTPAddr),
		JWTSecret:      getenv("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL: getenvDuration("ACCESS_TOKEN_TTL", defaultAccessTokenTTL),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
