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
	defaultReconnectTTL   = 5 * time.Minute
	defaultLogLevel       = "info"
)

type Config struct {
	AppEnv         string
	HTTPAddr       string
	RedisURL       string
	JWTSecret      string
	LogLevel       string
	AccessTokenTTL time.Duration
	ReconnectTTL   time.Duration
}

func LoadConfig() Config {
	return Config{
		AppEnv:         getenv("APP_ENV", defaultAppEnv),
		HTTPAddr:       getenv("HTTP_ADDR", defaultHTTPAddr),
		RedisURL:       getenv("REDIS_URL", ""),
		JWTSecret:      getenv("JWT_SECRET", defaultJWTSecret),
		LogLevel:       getenv("LOG_LEVEL", defaultLogLevel),
		AccessTokenTTL: getenvDuration("ACCESS_TOKEN_TTL", defaultAccessTokenTTL),
		ReconnectTTL:   getenvDuration("RECONNECT_TTL", defaultReconnectTTL),
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
