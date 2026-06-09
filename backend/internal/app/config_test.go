package app

import (
	"testing"
	"time"
)

func TestLoadConfigUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("ACCESS_TOKEN_TTL", "")
	t.Setenv("RECONNECT_TTL", "")

	cfg := LoadConfig()

	if cfg.AppEnv != "dev" {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, "dev")
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.RedisURL != "" {
		t.Fatalf("RedisURL = %q, want empty", cfg.RedisURL)
	}
	if cfg.JWTSecret != "change-me" {
		t.Fatalf("JWTSecret = %q, want %q", cfg.JWTSecret, "change-me")
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.AccessTokenTTL != 24*time.Hour {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 24*time.Hour)
	}
	if cfg.ReconnectTTL != 5*time.Minute {
		t.Fatalf("ReconnectTTL = %v, want %v", cfg.ReconnectTTL, 5*time.Minute)
	}
}

func TestLoadConfigReadsEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_ADDR", ":18080")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("ACCESS_TOKEN_TTL", "12h")
	t.Setenv("RECONNECT_TTL", "10m")

	cfg := LoadConfig()

	if cfg.AppEnv != "test" {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, "test")
	}
	if cfg.HTTPAddr != ":18080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":18080")
	}
	if cfg.RedisURL != "redis://127.0.0.1:6379/0" {
		t.Fatalf("RedisURL = %q, want %q", cfg.RedisURL, "redis://127.0.0.1:6379/0")
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-secret")
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.AccessTokenTTL != 12*time.Hour {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 12*time.Hour)
	}
	if cfg.ReconnectTTL != 10*time.Minute {
		t.Fatalf("ReconnectTTL = %v, want %v", cfg.ReconnectTTL, 10*time.Minute)
	}
}

func TestLoadConfigFallsBackWhenAccessTokenTTLInvalid(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_TTL", "bad-value")
	t.Setenv("RECONNECT_TTL", "bad-value")

	cfg := LoadConfig()

	if cfg.AccessTokenTTL != 24*time.Hour {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 24*time.Hour)
	}
	if cfg.ReconnectTTL != 5*time.Minute {
		t.Fatalf("ReconnectTTL = %v, want %v", cfg.ReconnectTTL, 5*time.Minute)
	}
}
