package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"ddz/backend/internal/auth"
	"ddz/backend/internal/room"
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
	roomManager    *room.Manager
}

func NewHTTPServer(cfg Config) *http.Server {
	return &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: NewHTTPHandler(cfg),
	}
}

func NewHTTPHandler(cfg Config) http.Handler {
	return NewHTTPHandlerWithManager(cfg, room.NewManager())
}

// NewHTTPHandlerWithManager 创建带有指定房间管理器的 HTTP Handler，便于测试和后续模块复用。
func NewHTTPHandlerWithManager(cfg Config, roomManager *room.Manager) http.Handler {
	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		panic(err)
	}
	if roomManager == nil {
		roomManager = room.NewManager()
	}

	app := &httpApp{
		authService:    auth.NewService(jwtManager),
		authMiddleware: auth.NewMiddleware(jwtManager),
		roomManager:    roomManager,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/api/v1/auth/guest", app.handleGuestLogin)
	mux.Handle("/api/v1/auth/me", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleCurrentUser)))
	mux.Handle("/api/v1/lobby/summary", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleLobbySummary)))
	mux.Handle("/api/v1/rooms", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleRoomList)))
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

func (a *httpApp) handleLobbySummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	summary, err := a.roomManager.GetLobbySummary()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code:      "ok",
		Message:   "ok",
		Data:      summary,
		RequestID: requestIDFromRequest(r),
	})
}

func (a *httpApp) handleRoomList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	filter, err := parseRoomListFilter(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "bad_request", "bad request")
		return
	}

	result, err := a.roomManager.ListRooms(filter)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code:      "ok",
		Message:   "ok",
		Data:      result,
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

func parseRoomListFilter(r *http.Request) (room.RoomListFilter, error) {
	query := r.URL.Query()

	page, err := parseOptionalPositiveInt(query.Get("page"))
	if err != nil {
		return room.RoomListFilter{}, err
	}
	pageSize, err := parseOptionalPositiveInt(query.Get("page_size"))
	if err != nil {
		return room.RoomListFilter{}, err
	}

	filter := room.RoomListFilter{
		Mode:     strings.TrimSpace(query.Get("mode")),
		Page:     page,
		PageSize: pageSize,
	}

	status := strings.TrimSpace(query.Get("status"))
	if status == "" {
		return filter, nil
	}

	parsedStatus, ok := parseRoomStatus(status)
	if !ok {
		return room.RoomListFilter{}, errors.New("invalid room status")
	}
	filter.Status = parsedStatus
	return filter, nil
}

func parseOptionalPositiveInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid integer query")
	}
	return parsed, nil
}

func parseRoomStatus(value string) (room.RoomStatus, bool) {
	switch room.RoomStatus(value) {
	case room.RoomStatusWaiting, room.RoomStatusPlaying, room.RoomStatusSettling, room.RoomStatusClosed:
		return room.RoomStatus(value), true
	default:
		return "", false
	}
}
