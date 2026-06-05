package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"ddz/backend/internal/auth"
)

type apiResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id,omitempty"`
}

type guestLoginRequest struct {
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type guestLoginResponseData struct {
	User        auth.User `json:"user"`
	AccessToken string    `json:"access_token"`
	ExpiresIn   int64     `json:"expires_in"`
}

type meResponseData struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"display_name"`
	AvatarURL   string       `json:"avatar_url"`
	AccountType string       `json:"account_type"`
	Profile     auth.Profile `json:"profile"`
}

type httpApp struct {
	authService    *auth.Service
	authMiddleware *auth.Middleware
}

func NewHTTPServer(cfg Config) *http.Server {
	return &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: NewHTTPHandler(cfg),
	}
}

func NewHTTPHandler(cfg Config) http.Handler {
	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		panic(err)
	}

	app := &httpApp{
		authService:    auth.NewService(jwtManager),
		authMiddleware: auth.NewMiddleware(jwtManager),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/api/v1/auth/guest", app.handleGuestLogin)
	mux.Handle("/api/v1/auth/me", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleCurrentUser)))
	return mux
}

func (a *httpApp) handleGuestLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	var req guestLoginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "bad_request", "bad request")
		return
	}

	result, err := a.authService.GuestLogin(auth.GuestLoginInput{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code:    "ok",
		Message: "ok",
		Data: guestLoginResponseData{
			User:        result.User,
			AccessToken: result.AccessToken,
			ExpiresIn:   result.ExpiresIn,
		},
		RequestID: requestIDFromRequest(r),
	})
}

func (a *httpApp) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	identity := auth.IdentityFromClaims(claims)
	writeJSON(w, http.StatusOK, apiResponse{
		Code:    "ok",
		Message: "ok",
		Data: meResponseData{
			ID:          identity.User.ID,
			DisplayName: identity.User.DisplayName,
			AvatarURL:   identity.User.AvatarURL,
			AccountType: identity.User.AccountType,
			Profile:     identity.Profile,
		},
		RequestID: requestIDFromRequest(r),
	})
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func decodeJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return nil
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload apiResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code string, message string) {
	writeJSON(w, statusCode, apiResponse{
		Code:      code,
		Message:   message,
		Data:      nil,
		RequestID: requestIDFromRequest(r),
	})
}

func requestIDFromRequest(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}
