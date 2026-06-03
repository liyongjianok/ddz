package app

import "os"

const (
	defaultAppEnv   = "dev"
	defaultHTTPAddr = ":8080"
)

type Config struct {
	AppEnv   string
	HTTPAddr string
}

func LoadConfig() Config {
	return Config{
		AppEnv:   getenv("APP_ENV", defaultAppEnv),
		HTTPAddr: getenv("HTTP_ADDR", defaultHTTPAddr),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
