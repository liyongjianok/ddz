package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"ddz/backend/internal/auth"
	"ddz/backend/internal/record"
	"ddz/backend/internal/room"
	"ddz/backend/internal/ws"
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

type quickStartRequest struct {
	Mode      string `json:"mode"`
	BaseScore int    `json:"base_score"`
}

type createRoomRequest struct {
	Mode      string `json:"mode"`
	BaseScore int    `json:"base_score"`
	Private   bool   `json:"private"`
}

type joinRoomRequest struct {
	PreferredSeat *int `json:"preferred_seat"`
}

type meResponseData struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"display_name"`
	AvatarURL   string       `json:"avatar_url"`
	AccountType string       `json:"account_type"`
	Profile     auth.Profile `json:"profile"`
}

type roomAccessResponseData struct {
	RoomID    string `json:"room_id"`
	SeatIndex int    `json:"seat_index"`
	WSURL     string `json:"ws_url"`
}

type leaveRoomResponseData struct {
	RoomID string `json:"room_id"`
	Left   bool   `json:"left"`
}

type recordsQuery struct {
	Page     int
	PageSize int
}

type httpApp struct {
	authService    *auth.Service
	authMiddleware *auth.Middleware
	roomManager    *room.Manager
	recordService  *record.Service
}

func NewHTTPServer(cfg Config) *http.Server {
	return &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: NewHTTPHandler(cfg),
	}
}

func NewHTTPHandler(cfg Config) http.Handler {
	return NewHTTPHandlerWithDependencies(
		cfg,
		room.NewManager(),
		record.NewService(NewRecordStore(cfg, NewRedisStore(cfg))),
	)
}

// NewHTTPHandlerWithManager 创建带指定房间管理器的 HTTP Handler，便于测试和复用。
func NewHTTPHandlerWithManager(cfg Config, roomManager *room.Manager) http.Handler {
	return NewHTTPHandlerWithDependencies(
		cfg,
		roomManager,
		record.NewService(NewRecordStore(cfg, NewRedisStore(cfg))),
	)
}

// NewHTTPHandlerWithDependencies 创建带显式依赖的 HTTP Handler。
func NewHTTPHandlerWithDependencies(cfg Config, roomManager *room.Manager, recordService *record.Service) http.Handler {
	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		panic(err)
	}
	if roomManager == nil {
		roomManager = room.NewManager()
	}
	if recordService == nil {
		recordService = record.NewService(NewRecordStore(cfg, NewRedisStore(cfg)))
	}

	app := &httpApp{
		authService:    auth.NewService(jwtManager),
		authMiddleware: auth.NewMiddleware(jwtManager),
		roomManager:    roomManager,
		recordService:  recordService,
	}
	wsGateway := ws.NewGateway(jwtManager, roomManager, recordService)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/api/v1/auth/guest", app.handleGuestLogin)
	mux.Handle("/api/v1/auth/me", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleCurrentUser)))
	mux.Handle("/api/v1/lobby/summary", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleLobbySummary)))
	mux.Handle("/api/v1/matchmaking/quick-start", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleQuickStart)))
	mux.Handle("/api/v1/records/my", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleMyRecords)))
	mux.Handle("/api/v1/records/", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleRecordDetail)))
	mux.Handle("/api/v1/rooms", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleRooms)))
	mux.Handle("/api/v1/rooms/", app.authMiddleware.RequireAuth(http.HandlerFunc(app.handleRoomActions)))
	mux.Handle("/ws/v1/rooms/", wsGateway)
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

func (a *httpApp) handleQuickStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	var req quickStartRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "bad_request", "bad request")
		return
	}

	userID, ok := authenticatedUserID(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	currentRoom, seatIndex, err := a.roomManager.QuickStart(room.QuickStartInput{
		UserID:    userID,
		Mode:      req.Mode,
		BaseScore: req.BaseScore,
	})
	if err != nil {
		writeRoomError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code:    "ok",
		Message: "ok",
		Data: roomAccessResponseData{
			RoomID:    currentRoom.ID,
			SeatIndex: seatIndex,
			WSURL:     buildRoomWSURL(currentRoom.ID),
		},
		RequestID: requestIDFromRequest(r),
	})
}

func (a *httpApp) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleRoomList(w, r)
	case http.MethodPost:
		a.handleCreateRoom(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
	}
}

func (a *httpApp) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "bad_request", "bad request")
		return
	}

	userID, ok := authenticatedUserID(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	currentRoom, seatIndex, err := a.roomManager.CreateRoom(room.CreateRoomInput{
		UserID:    userID,
		Mode:      req.Mode,
		BaseScore: req.BaseScore,
	})
	if err != nil {
		writeRoomError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code:    "ok",
		Message: "ok",
		Data: roomAccessResponseData{
			RoomID:    currentRoom.ID,
			SeatIndex: seatIndex,
			WSURL:     buildRoomWSURL(currentRoom.ID),
		},
		RequestID: requestIDFromRequest(r),
	})
}

func (a *httpApp) handleRoomActions(w http.ResponseWriter, r *http.Request) {
	roomID, action, ok := parseRoomActionPath(r.URL.Path)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found", "not found")
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	switch action {
	case "join":
		a.handleJoinRoom(w, r, roomID)
	case "leave":
		a.handleLeaveRoom(w, r, roomID)
	default:
		writeError(w, r, http.StatusNotFound, "not_found", "not found")
	}
}

func (a *httpApp) handleJoinRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	var req joinRoomRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "bad_request", "bad request")
		return
	}

	userID, ok := authenticatedUserID(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	currentRoom, seatIndex, err := a.roomManager.JoinRoom(room.JoinRoomInput{
		RoomID:        roomID,
		UserID:        userID,
		PreferredSeat: req.PreferredSeat,
	})
	if err != nil {
		writeRoomError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code:    "ok",
		Message: "ok",
		Data: roomAccessResponseData{
			RoomID:    currentRoom.ID,
			SeatIndex: seatIndex,
			WSURL:     buildRoomWSURL(currentRoom.ID),
		},
		RequestID: requestIDFromRequest(r),
	})
}

func (a *httpApp) handleLeaveRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	userID, ok := authenticatedUserID(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	if _, err := a.roomManager.LeaveRoom(roomID, userID); err != nil {
		writeRoomError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Code:    "ok",
		Message: "ok",
		Data: leaveRoomResponseData{
			RoomID: roomID,
			Left:   true,
		},
		RequestID: requestIDFromRequest(r),
	})
}

func (a *httpApp) handleMyRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	userID, ok := authenticatedUserID(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	query, err := parseRecordsQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "bad_request", "bad request")
		return
	}

	result, err := a.recordService.ListMyRecords(r.Context(), userID, query.Page, query.PageSize)
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

func (a *httpApp) handleRecordDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	userID, ok := authenticatedUserID(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	gameID, ok := parseRecordGameID(r.URL.Path)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found", "not found")
		return
	}

	result, err := a.recordService.GetGameRecord(r.Context(), userID, gameID)
	if err != nil {
		writeRecordError(w, r, err)
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

func parseRecordsQuery(r *http.Request) (recordsQuery, error) {
	query := r.URL.Query()

	page, err := parseOptionalPositiveInt(query.Get("page"))
	if err != nil {
		return recordsQuery{}, err
	}
	pageSize, err := parseOptionalPositiveInt(query.Get("page_size"))
	if err != nil {
		return recordsQuery{}, err
	}

	return recordsQuery{
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func parseRoomStatus(value string) (room.RoomStatus, bool) {
	switch room.RoomStatus(value) {
	case room.RoomStatusWaiting, room.RoomStatusPlaying, room.RoomStatusSettling, room.RoomStatusClosed:
		return room.RoomStatus(value), true
	default:
		return "", false
	}
}

func authenticatedUserID(r *http.Request) (string, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.Subject == "" {
		return "", false
	}
	return claims.Subject, true
}

func parseRoomActionPath(path string) (string, string, bool) {
	const prefix = "/api/v1/rooms/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseRecordGameID(path string) (string, bool) {
	const prefix = "/api/v1/records/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	gameID := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if gameID == "" || gameID == "my" || strings.Contains(gameID, "/") {
		return "", false
	}
	return gameID, true
}

func buildRoomWSURL(roomID string) string {
	return "/ws/v1/rooms/" + roomID
}

func writeRoomError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode, code, message := mapRoomError(err)
	writeError(w, r, statusCode, code, message)
}

func mapRoomError(err error) (int, string, string) {
	switch {
	case errors.Is(err, room.ErrInvalidRoomConfig):
		return http.StatusBadRequest, "bad_request", "bad request"
	case errors.Is(err, room.ErrRoomNotFound), errors.Is(err, room.ErrRoomClosed):
		return http.StatusNotFound, "room_not_found", "room not found"
	case errors.Is(err, room.ErrSeatUnavailable):
		return http.StatusBadRequest, "seat_unavailable", "seat unavailable"
	case errors.Is(err, room.ErrUserAlreadyInActiveRoom):
		return http.StatusConflict, "already_in_room", "already in room"
	case errors.Is(err, room.ErrGameAlreadyStarted):
		return http.StatusConflict, "game_already_started", "game already started"
	case errors.Is(err, room.ErrUserNotInRoom):
		return http.StatusBadRequest, "not_in_room", "not in room"
	case errors.Is(err, room.ErrRoomFull):
		return http.StatusConflict, "bad_request", "room full"
	default:
		return http.StatusInternalServerError, "internal_error", "internal error"
	}
}

func writeRecordError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode, code, message := mapRecordError(err)
	writeError(w, r, statusCode, code, message)
}

func mapRecordError(err error) (int, string, string) {
	switch {
	case errors.Is(err, record.ErrRecordNotFound):
		return http.StatusNotFound, "not_found", "not found"
	case errors.Is(err, record.ErrRecordForbidden):
		return http.StatusForbidden, "forbidden", "forbidden"
	default:
		return http.StatusInternalServerError, "internal_error", "internal error"
	}
}
