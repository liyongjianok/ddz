package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ddz/backend/internal/game"
	"ddz/backend/internal/record"
	"ddz/backend/internal/room"
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

type lobbySummaryData struct {
	OnlinePlayers int `json:"online_players"`
	ActiveRooms   int `json:"active_rooms"`
	Modes         []struct {
		Mode          string `json:"mode"`
		BaseScore     int    `json:"base_score"`
		OnlinePlayers int    `json:"online_players"`
		WaitingRooms  int    `json:"waiting_rooms"`
	} `json:"modes"`
}

type roomListData struct {
	Items []struct {
		RoomID      string    `json:"room_id"`
		Mode        string    `json:"mode"`
		Status      string    `json:"status"`
		BaseScore   int       `json:"base_score"`
		PlayerCount int       `json:"player_count"`
		MaxPlayers  int       `json:"max_players"`
		CreatedAt   time.Time `json:"created_at"`
	} `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type roomAccessData struct {
	RoomID    string `json:"room_id"`
	SeatIndex int    `json:"seat_index"`
	WSURL     string `json:"ws_url"`
}

type leaveRoomData struct {
	RoomID string `json:"room_id"`
	Left   bool   `json:"left"`
}

type recordListData struct {
	Items []struct {
		GameID     string    `json:"game_id"`
		Mode       string    `json:"mode"`
		Role       string    `json:"role"`
		WinnerSide string    `json:"winner_side"`
		ScoreDelta int       `json:"score_delta"`
		StartedAt  time.Time `json:"started_at"`
		EndedAt    time.Time `json:"ended_at"`
	} `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type recordDetailData struct {
	GameID     string                `json:"game_id"`
	RoomID     string                `json:"room_id"`
	Mode       string                `json:"mode"`
	BaseScore  int                   `json:"base_score"`
	Multiplier int                   `json:"multiplier"`
	WinnerSide string                `json:"winner_side"`
	Players    []record.RecordPlayer `json:"players"`
	Events     []record.Event        `json:"events"`
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

func TestLobbySummaryReturnsAggregatesWithValidToken(t *testing.T) {
	manager := newLobbyTestManager(t)
	handler := NewHTTPHandlerWithManager(testConfig(), manager)
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lobby/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "req_lobby_1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var envelope apiResponseEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != "ok" {
		t.Fatalf("code = %q, want %q", envelope.Code, "ok")
	}
	if envelope.RequestID != "req_lobby_1" {
		t.Fatalf("request id = %q, want %q", envelope.RequestID, "req_lobby_1")
	}

	var data lobbySummaryData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.OnlinePlayers != 6 {
		t.Fatalf("online players = %d, want 6", data.OnlinePlayers)
	}
	if data.ActiveRooms != 3 {
		t.Fatalf("active rooms = %d, want 3", data.ActiveRooms)
	}
	if len(data.Modes) != 2 {
		t.Fatalf("mode summary len = %d, want 2", len(data.Modes))
	}
}

func TestRoomListReturnsPublicFieldsOnly(t *testing.T) {
	manager := newLobbyTestManager(t)
	handler := NewHTTPHandlerWithManager(testConfig(), manager)
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms?mode=classic&status=waiting&page=1&page_size=20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if strings.Contains(body, `"hand"`) || strings.Contains(body, `"current_game"`) || strings.Contains(body, `"bottom_cards"`) {
		t.Fatalf("room list leaked hidden game fields: %s", body)
	}

	var envelope apiResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var data roomListData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.Total != 2 {
		t.Fatalf("total = %d, want 2", data.Total)
	}
	if len(data.Items) != 2 {
		t.Fatalf("item len = %d, want 2", len(data.Items))
	}
	for _, item := range data.Items {
		if item.Mode != "classic" {
			t.Fatalf("mode = %q, want %q", item.Mode, "classic")
		}
		if item.Status != "waiting" {
			t.Fatalf("status = %q, want %q", item.Status, "waiting")
		}
	}
}

func TestRoomListRejectsInvalidQuery(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManager())
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms?page=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestQuickStartReturnsRoomAndWSURL(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	rec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/matchmaking/quick-start", token, `{"mode":"classic","base_score":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	envelope := decodeResponseEnvelope(t, rec)
	if envelope.Code != "ok" {
		t.Fatalf("code = %q, want %q", envelope.Code, "ok")
	}

	var data roomAccessData
	decodeResponseData(t, envelope, &data)
	if data.RoomID == "" {
		t.Fatal("room_id should not be empty")
	}
	if data.SeatIndex != 0 {
		t.Fatalf("seat_index = %d, want %d", data.SeatIndex, 0)
	}
	if data.WSURL != "/ws/v1/rooms/"+data.RoomID {
		t.Fatalf("ws_url = %q, want %q", data.WSURL, "/ws/v1/rooms/"+data.RoomID)
	}
}

func TestQuickStartReturnsExistingRoomForSameUser(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	firstRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/matchmaking/quick-start", token, `{"mode":"classic","base_score":1}`)
	secondRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/matchmaking/quick-start", token, `{"mode":"classic","base_score":1}`)

	var firstData roomAccessData
	decodeResponseData(t, decodeResponseEnvelope(t, firstRec), &firstData)

	var secondData roomAccessData
	decodeResponseData(t, decodeResponseEnvelope(t, secondRec), &secondData)

	if secondData.RoomID != firstData.RoomID {
		t.Fatalf("room_id = %q, want %q", secondData.RoomID, firstData.RoomID)
	}
	if secondData.SeatIndex != firstData.SeatIndex {
		t.Fatalf("seat_index = %d, want %d", secondData.SeatIndex, firstData.SeatIndex)
	}
	if secondData.WSURL != firstData.WSURL {
		t.Fatalf("ws_url = %q, want %q", secondData.WSURL, firstData.WSURL)
	}
}

func TestCreateRoomReturnsRoomAndWSURL(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	rec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms", token, `{"mode":"classic","base_score":2,"private":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var data roomAccessData
	decodeResponseData(t, decodeResponseEnvelope(t, rec), &data)
	if data.RoomID == "" {
		t.Fatal("room_id should not be empty")
	}
	if data.SeatIndex != 0 {
		t.Fatalf("seat_index = %d, want %d", data.SeatIndex, 0)
	}
	if data.WSURL != "/ws/v1/rooms/"+data.RoomID {
		t.Fatalf("ws_url = %q, want %q", data.WSURL, "/ws/v1/rooms/"+data.RoomID)
	}
}

func TestJoinRoomReturnsPreferredSeatAndWSURL(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	hostToken := loginAndGetToken(t, handler, `{"display_name":"Host"}`)
	guestToken := loginAndGetToken(t, handler, `{"display_name":"Guest"}`)

	createRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms", hostToken, `{"mode":"classic","base_score":1,"private":false}`)
	var created roomAccessData
	decodeResponseData(t, decodeResponseEnvelope(t, createRec), &created)

	joinRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms/"+created.RoomID+"/join", guestToken, `{"preferred_seat":2}`)
	if joinRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", joinRec.Code, http.StatusOK)
	}

	var joined roomAccessData
	decodeResponseData(t, decodeResponseEnvelope(t, joinRec), &joined)
	if joined.RoomID != created.RoomID {
		t.Fatalf("room_id = %q, want %q", joined.RoomID, created.RoomID)
	}
	if joined.SeatIndex != 2 {
		t.Fatalf("seat_index = %d, want %d", joined.SeatIndex, 2)
	}
	if joined.WSURL != "/ws/v1/rooms/"+created.RoomID {
		t.Fatalf("ws_url = %q, want %q", joined.WSURL, "/ws/v1/rooms/"+created.RoomID)
	}
}

func TestLeaveRoomReturnsLeftTrue(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	createRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms", token, `{"mode":"classic","base_score":1,"private":false}`)
	var created roomAccessData
	decodeResponseData(t, decodeResponseEnvelope(t, createRec), &created)

	leaveRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms/"+created.RoomID+"/leave", token, "")
	if leaveRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", leaveRec.Code, http.StatusOK)
	}

	var left leaveRoomData
	decodeResponseData(t, decodeResponseEnvelope(t, leaveRec), &left)
	if left.RoomID != created.RoomID {
		t.Fatalf("room_id = %q, want %q", left.RoomID, created.RoomID)
	}
	if !left.Left {
		t.Fatal("left should be true")
	}
}

func TestJoinRoomRejectsRoomNotFound(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	rec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms/r_missing/join", token, `{"preferred_seat":1}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	envelope := decodeResponseEnvelope(t, rec)
	if envelope.Code != "room_not_found" {
		t.Fatalf("code = %q, want %q", envelope.Code, "room_not_found")
	}
}

func TestMyRecordsReturnsCompletedGames(t *testing.T) {
	recordService := record.NewService(record.NewMemoryStore())
	handler := NewHTTPHandlerWithDependencies(testConfig(), room.NewManager(), recordService)
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)
	user := currentUserWithToken(t, handler, token)

	err := recordService.AppendEvent(nilContext(), record.Event{
		GameID:      "g_001",
		RoomID:      "r_001",
		Seq:         1,
		EventType:   "game_ended",
		ActorUserID: user.ID,
		Payload:     map[string]any{"winner_side": "landlord"},
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}
	err = recordService.SaveGameRecord(nilContext(), &room.Room{
		ID:        "r_001",
		Mode:      "classic",
		BaseScore: 1,
		CurrentGame: &game.Game{
			ID:        "g_001",
			StartedAt: time.Now().UTC().Add(-time.Minute),
			EndedAt:   time.Now().UTC(),
			Players: []game.PlayerState{
				{UserID: user.ID, SeatIndex: 0, Role: game.RoleLandlord},
				{UserID: "u2", SeatIndex: 1, Role: game.RoleFarmer},
				{UserID: "u3", SeatIndex: 2, Role: game.RoleFarmer},
			},
			Settlement: &game.SettlementResult{
				BaseScore:  1,
				Multiplier: 3,
				WinnerSide: game.WinnerSideLandlord,
				Players: []game.SettlementPlayer{
					{UserID: user.ID, SeatIndex: 0, Role: game.RoleLandlord, DeltaScore: 6, IsWinner: true},
					{UserID: "u2", SeatIndex: 1, Role: game.RoleFarmer, DeltaScore: -3},
					{UserID: "u3", SeatIndex: 2, Role: game.RoleFarmer, DeltaScore: -3},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveGameRecord() error = %v", err)
	}

	rec := doAuthenticatedJSONRequest(t, handler, http.MethodGet, "/api/v1/records/my?page=1&page_size=20", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var data recordListData
	decodeResponseData(t, decodeResponseEnvelope(t, rec), &data)
	if data.Total != 1 {
		t.Fatalf("total = %d, want 1", data.Total)
	}
	if len(data.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(data.Items))
	}
	if data.Items[0].GameID != "g_001" {
		t.Fatalf("game_id = %q, want %q", data.Items[0].GameID, "g_001")
	}
}

func TestRecordDetailRejectsNonParticipant(t *testing.T) {
	recordService := record.NewService(record.NewMemoryStore())
	handler := NewHTTPHandlerWithDependencies(testConfig(), room.NewManager(), recordService)
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	err := recordService.SaveGameRecord(nilContext(), &room.Room{
		ID:        "r_001",
		Mode:      "classic",
		BaseScore: 1,
		CurrentGame: &game.Game{
			ID:        "g_001",
			StartedAt: time.Now().UTC().Add(-time.Minute),
			EndedAt:   time.Now().UTC(),
			Players: []game.PlayerState{
				{UserID: "u1", SeatIndex: 0, Role: game.RoleLandlord},
				{UserID: "u2", SeatIndex: 1, Role: game.RoleFarmer},
				{UserID: "u3", SeatIndex: 2, Role: game.RoleFarmer},
			},
			Settlement: &game.SettlementResult{
				BaseScore:  1,
				Multiplier: 3,
				WinnerSide: game.WinnerSideLandlord,
				Players: []game.SettlementPlayer{
					{UserID: "u1", SeatIndex: 0, Role: game.RoleLandlord, DeltaScore: 6, IsWinner: true},
					{UserID: "u2", SeatIndex: 1, Role: game.RoleFarmer, DeltaScore: -3},
					{UserID: "u3", SeatIndex: 2, Role: game.RoleFarmer, DeltaScore: -3},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveGameRecord() error = %v", err)
	}

	rec := doAuthenticatedJSONRequest(t, handler, http.MethodGet, "/api/v1/records/g_001", token, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCreateRoomRejectsAlreadyInRoom(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)

	firstRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms", token, `{"mode":"classic","base_score":1,"private":false}`)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first create status = %d, want %d", firstRec.Code, http.StatusOK)
	}

	secondRec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms", token, `{"mode":"classic","base_score":1,"private":false}`)
	if secondRec.Code != http.StatusConflict {
		t.Fatalf("second create status = %d, want %d", secondRec.Code, http.StatusConflict)
	}

	envelope := decodeResponseEnvelope(t, secondRec)
	if envelope.Code != "already_in_room" {
		t.Fatalf("code = %q, want %q", envelope.Code, "already_in_room")
	}
}

func nilContext() context.Context {
	return context.Background()
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

func doAuthenticatedJSONRequest(t *testing.T, handler http.Handler, method string, path string, token string, payload string) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if payload != "" {
		body = strings.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, body)
	if payload != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeResponseEnvelope(t *testing.T, rec *httptest.ResponseRecorder) apiResponseEnvelope {
	t.Helper()

	var envelope apiResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return envelope
}

func decodeResponseData(t *testing.T, envelope apiResponseEnvelope, target any) {
	t.Helper()

	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode response data: %v", err)
	}
}

func testConfig() Config {
	return Config{
		AppEnv:         "test",
		HTTPAddr:       ":18080",
		JWTSecret:      "test-secret",
		AccessTokenTTL: 24 * time.Hour,
	}
}

func newLobbyTestManager(t *testing.T) *room.Manager {
	t.Helper()

	manager := room.NewManagerWithRNG(&fixedRoomRNG{value: 0})

	roomA, _, err := manager.CreateRoom(room.CreateRoomInput{UserID: "u1", BaseScore: 1, Mode: "classic"})
	if err != nil {
		t.Fatalf("CreateRoom roomA error = %v", err)
	}
	if _, _, err := manager.JoinRoom(room.JoinRoomInput{RoomID: roomA.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom roomA error = %v", err)
	}

	if _, _, err := manager.CreateRoom(room.CreateRoomInput{UserID: "u3", BaseScore: 1, Mode: "classic"}); err != nil {
		t.Fatalf("CreateRoom roomB error = %v", err)
	}

	roomC, _, err := manager.CreateRoom(room.CreateRoomInput{UserID: "u4", BaseScore: 2, Mode: "classic"})
	if err != nil {
		t.Fatalf("CreateRoom roomC error = %v", err)
	}
	if _, _, err := manager.JoinRoom(room.JoinRoomInput{RoomID: roomC.ID, UserID: "u5"}); err != nil {
		t.Fatalf("JoinRoom roomC u5 error = %v", err)
	}
	if _, _, err := manager.JoinRoom(room.JoinRoomInput{RoomID: roomC.ID, UserID: "u6"}); err != nil {
		t.Fatalf("JoinRoom roomC u6 error = %v", err)
	}
	if _, _, _, err := manager.Ready(room.ReadyInput{RoomID: roomC.ID, UserID: "u4", Ready: true}); err != nil {
		t.Fatalf("Ready roomC u4 error = %v", err)
	}
	if _, _, _, err := manager.Ready(room.ReadyInput{RoomID: roomC.ID, UserID: "u5", Ready: true}); err != nil {
		t.Fatalf("Ready roomC u5 error = %v", err)
	}
	if _, _, _, err := manager.Ready(room.ReadyInput{RoomID: roomC.ID, UserID: "u6", Ready: true}); err != nil {
		t.Fatalf("Ready roomC u6 error = %v", err)
	}

	return manager
}

type fixedRoomRNG struct {
	value int
}

func (r *fixedRoomRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	value := r.value % n
	if value < 0 {
		value += n
	}
	return value
}
