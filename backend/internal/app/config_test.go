package app

import (
	"testing"
	"time"
)

func TestLoadConfigUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ACCESS_TOKEN_TTL", "")

	cfg := LoadConfig()

	if cfg.AppEnv != "dev" {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, "dev")
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.JWTSecret != "change-me" {
		t.Fatalf("JWTSecret = %q, want %q", cfg.JWTSecret, "change-me")
	}
	if cfg.AccessTokenTTL != 24*time.Hour {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 24*time.Hour)
	}
}

func TestLoadConfigReadsEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_ADDR", ":18080")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ACCESS_TOKEN_TTL", "12h")

	cfg := LoadConfig()

	if cfg.AppEnv != "test" {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, "test")
	}
	if cfg.HTTPAddr != ":18080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":18080")
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-secret")
	}
	if cfg.AccessTokenTTL != 12*time.Hour {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 12*time.Hour)
	}
}

func TestLoadConfigFallsBackWhenAccessTokenTTLInvalid(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_TTL", "bad-value")

	cfg := LoadConfig()

	if cfg.AccessTokenTTL != 24*time.Hour {
		t.Fatalf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 24*time.Hour)
	}
}
