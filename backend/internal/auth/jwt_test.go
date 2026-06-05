package auth

import (
	"errors"
	"testing"
	"time"
)

func TestJWTManagerIssueAndParse(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	manager, err := newJWTManagerWithClock("secret", 2*time.Hour, func() time.Time {
		return now
	})
	if err != nil {
		t.Fatalf("newJWTManagerWithClock() error = %v", err)
	}

	token, claims, err := manager.Issue(User{
		ID:          "u_000001",
		DisplayName: "Guest001",
		AvatarURL:   "",
		AccountType: AccountTypeGuest,
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}
	if claims.Subject != "u_000001" {
		t.Fatalf("claims subject = %q, want %q", claims.Subject, "u_000001")
	}
	if claims.ExpiresAt-claims.IssuedAt != int64((2*time.Hour)/time.Second) {
		t.Fatalf("ttl seconds = %d, want %d", claims.ExpiresAt-claims.IssuedAt, int64((2*time.Hour)/time.Second))
	}

	parsed, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.DisplayName != "Guest001" {
		t.Fatalf("display name = %q, want %q", parsed.DisplayName, "Guest001")
	}
}

func TestJWTManagerRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	current := now
	manager, err := newJWTManagerWithClock("secret", time.Hour, func() time.Time {
		return current
	})
	if err != nil {
		t.Fatalf("newJWTManagerWithClock() error = %v", err)
	}

	token, _, err := manager.Issue(User{
		ID:          "u_000001",
		DisplayName: "Guest001",
		AccountType: AccountTypeGuest,
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	current = now.Add(2 * time.Hour)
	_, err = manager.Parse(token)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("Parse() error = %v, want ErrTokenExpired", err)
	}
}

func TestJWTManagerRejectsTamperedToken(t *testing.T) {
	manager, err := NewJWTManager("secret", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	token, _, err := manager.Issue(User{
		ID:          "u_000001",
		DisplayName: "Guest001",
		AccountType: AccountTypeGuest,
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	tampered := token + "x"
	_, err = manager.Parse(tampered)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
	}
}
