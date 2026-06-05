package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddlewareRequireAuthAcceptsValidToken(t *testing.T) {
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

	var got Claims
	handler := NewMiddleware(manager).RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("claims should exist in context")
		}
		got = claims
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got.Subject != "u_000001" {
		t.Fatalf("subject = %q, want %q", got.Subject, "u_000001")
	}
}

func TestMiddlewareRequireAuthRejectsInvalidToken(t *testing.T) {
	manager, err := NewJWTManager("secret", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	handler := NewMiddleware(manager).RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer bad.token")
	req.Header.Set("X-Request-ID", "req_auth_1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var payload errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload.Code != "unauthorized" {
		t.Fatalf("code = %q, want %q", payload.Code, "unauthorized")
	}
	if payload.RequestID != "req_auth_1" {
		t.Fatalf("request id = %q, want %q", payload.RequestID, "req_auth_1")
	}
}
