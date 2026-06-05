package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type apiResponseEnvelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id"`
}

type guestLoginData struct {
	User struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
		AccountType string `json:"account_type"`
	} `json:"user"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type currentUserData struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	AccountType string `json:"account_type"`
	Profile     struct {
		Level       int `json:"level"`
		CoinBalance int `json:"coin_balance"`
		TotalGames  int `json:"total_games"`
		Wins        int `json:"wins"`
	} `json:"profile"`
}

func TestHealthzReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	NewHTTPHandler(testConfig()).ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "ok\n" {
		t.Fatalf("body = %q, want %q", string(body), "ok\n")
	}
}

func TestGuestLoginReturnsUserAndAccessToken(t *testing.T) {
	handler := NewHTTPHandler(testConfig())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/guest", strings.NewReader(`{"display_name":"Guest123","avatar_url":"https://example.com/a.png"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req_guest_1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var envelope apiResponseEnvelope
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != "ok" {
		t.Fatalf("code = %q, want %q", envelope.Code, "ok")
	}
	if envelope.RequestID != "req_guest_1" {
		t.Fatalf("request id = %q, want %q", envelope.RequestID, "req_guest_1")
	}

	var data guestLoginData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.User.ID == "" {
		t.Fatal("user id should not be empty")
	}
	if data.User.DisplayName != "Guest123" {
		t.Fatalf("display name = %q, want %q", data.User.DisplayName, "Guest123")
	}
	if data.User.AvatarURL != "https://example.com/a.png" {
		t.Fatalf("avatar url = %q, want %q", data.User.AvatarURL, "https://example.com/a.png")
	}
	if data.User.AccountType != "guest" {
		t.Fatalf("account type = %q, want %q", data.User.AccountType, "guest")
	}
	if data.AccessToken == "" {
		t.Fatal("access token should not be empty")
	}
	if data.ExpiresIn != 86400 {
		t.Fatalf("expires_in = %d, want %d", data.ExpiresIn, 86400)
	}
}

func TestGuestLoginGeneratesDefaultDisplayNameWhenEmpty(t *testing.T) {
	handler := NewHTTPHandler(testConfig())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/guest", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var envelope apiResponseEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var data guestLoginData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if !strings.HasPrefix(data.User.DisplayName, "Guest") {
		t.Fatalf("display name = %q, want prefix %q", data.User.DisplayName, "Guest")
	}
}

func TestCurrentUserReturnsIdentityWithValidToken(t *testing.T) {
	handler := NewHTTPHandler(testConfig())
	token := loginAndGetToken(t, handler, `{"display_name":"A","avatar_url":""}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "req_me_1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var envelope apiResponseEnvelope
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != "ok" {
		t.Fatalf("code = %q, want %q", envelope.Code, "ok")
	}
	if envelope.RequestID != "req_me_1" {
		t.Fatalf("request id = %q, want %q", envelope.RequestID, "req_me_1")
	}

	var data currentUserData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.DisplayName != "A" {
		t.Fatalf("display name = %q, want %q", data.DisplayName, "A")
	}
	if data.AccountType != "guest" {
		t.Fatalf("account type = %q, want %q", data.AccountType, "guest")
	}
	if data.Profile.Level != 1 {
		t.Fatalf("profile level = %d, want 1", data.Profile.Level)
	}
	if data.Profile.CoinBalance != 10000 {
		t.Fatalf("coin balance = %d, want 10000", data.Profile.CoinBalance)
	}
}

func TestCurrentUserRejectsInvalidToken(t *testing.T) {
	handler := NewHTTPHandler(testConfig())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.value")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}

	var envelope apiResponseEnvelope
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != "unauthorized" {
		t.Fatalf("code = %q, want %q", envelope.Code, "unauthorized")
	}
}

func TestCurrentUserRejectsMissingToken(t *testing.T) {
	handler := NewHTTPHandler(testConfig())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func loginAndGetToken(t *testing.T, handler http.Handler, payload string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/guest", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("guest login status = %d, want %d", rec.Code, http.StatusOK)
	}

	var envelope apiResponseEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode guest login response: %v", err)
	}

	var data guestLoginData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode guest login data: %v", err)
	}
	return data.AccessToken
}

func testConfig() Config {
	return Config{
		AppEnv:         "test",
		HTTPAddr:       ":18080",
		JWTSecret:      "test-secret",
		AccessTokenTTL: 24 * time.Hour,
	}
}
