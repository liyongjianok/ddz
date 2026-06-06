package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"ddz/backend/internal/room"

	"github.com/gorilla/websocket"
)

type wsServerEnvelope struct {
	Type       string          `json:"type"`
	RequestID  *string         `json:"request_id"`
	Seq        uint64          `json:"seq"`
	ServerTime time.Time       `json:"server_time"`
	Payload    json.RawMessage `json:"payload"`
}

type wsAckPayload struct {
	Ok bool `json:"ok"`
}

type wsErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type wsTestUser struct {
	ID    string
	Token string
}

type wsTestRoom struct {
	Manager *room.Manager
	Handler http.Handler
	RoomID  string
	Host    wsTestUser
	User2   wsTestUser
	User3   wsTestUser
}

func TestRoomWebSocketConnectSendsSnapshot(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)
	access := createRoomViaAPI(t, handler, token, `{"mode":"classic","base_score":1,"private":false}`)

	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(buildRoomWSConnectURL(server.URL, access.RoomID, token), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	var envelope wsServerEnvelope
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	if envelope.Type != "room.snapshot" {
		t.Fatalf("type = %q, want %q", envelope.Type, "room.snapshot")
	}
	if envelope.RequestID != nil {
		t.Fatalf("request_id = %v, want nil", *envelope.RequestID)
	}
	if envelope.Seq != 1 {
		t.Fatalf("seq = %d, want 1", envelope.Seq)
	}
	if envelope.ServerTime.IsZero() {
		t.Fatal("server_time should not be zero")
	}

	var snapshot room.RoomSnapshot
	if err := json.Unmarshal(envelope.Payload, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if snapshot.Room.RoomID != access.RoomID {
		t.Fatalf("snapshot room_id = %q, want %q", snapshot.Room.RoomID, access.RoomID)
	}
	if snapshot.Room.Status != string(room.RoomStatusWaiting) {
		t.Fatalf("snapshot status = %q, want %q", snapshot.Room.Status, room.RoomStatusWaiting)
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("player len = %d, want 1", len(snapshot.Players))
	}
	if snapshot.Me.SeatIndex != access.SeatIndex {
		t.Fatalf("me seat_index = %d, want %d", snapshot.Me.SeatIndex, access.SeatIndex)
	}
	if len(snapshot.Me.Hand) != 0 {
		t.Fatalf("me hand len = %d, want 0", len(snapshot.Me.Hand))
	}
	if snapshot.Game != nil {
		t.Fatal("game should be nil before room starts")
	}
}

func TestRoomWebSocketAcceptsAuthorizationHeader(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)
	access := createRoomViaAPI(t, handler, token, `{"mode":"classic","base_score":1,"private":false}`)

	server := httptest.NewServer(handler)
	defer server.Close()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, _, err := websocket.DefaultDialer.Dial(buildRoomWSBaseURL(server.URL, access.RoomID), header)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	var envelope wsServerEnvelope
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if envelope.Type != "room.snapshot" {
		t.Fatalf("type = %q, want %q", envelope.Type, "room.snapshot")
	}
}

func TestRoomWebSocketRejectsInvalidToken(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	server := httptest.NewServer(handler)
	defer server.Close()

	_, resp, err := websocket.DefaultDialer.Dial(buildRoomWSConnectURL(server.URL, "r_000001", "invalid.token.value"), nil)
	if err == nil {
		t.Fatal("Dial() error should not be nil")
	}
	if resp == nil {
		t.Fatalf("Dial() response should not be nil: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	var envelope apiResponseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != "unauthorized" {
		t.Fatalf("code = %q, want %q", envelope.Code, "unauthorized")
	}
}

func TestRoomWebSocketRejectsUserNotInRoom(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	hostToken := loginAndGetToken(t, handler, `{"display_name":"Host"}`)
	guestToken := loginAndGetToken(t, handler, `{"display_name":"Guest"}`)
	access := createRoomViaAPI(t, handler, hostToken, `{"mode":"classic","base_score":1,"private":false}`)

	server := httptest.NewServer(handler)
	defer server.Close()

	_, resp, err := websocket.DefaultDialer.Dial(buildRoomWSConnectURL(server.URL, access.RoomID, guestToken), nil)
	if err == nil {
		t.Fatal("Dial() error should not be nil")
	}
	if resp == nil {
		t.Fatalf("Dial() response should not be nil: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("read body: %v", readErr)
	}

	var envelope apiResponseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(body))
	}
	if envelope.Code != "not_in_room" {
		t.Fatalf("code = %q, want %q", envelope.Code, "not_in_room")
	}
}

func TestRoomWebSocketPingReturnsPong(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)
	access := createRoomViaAPI(t, handler, token, `{"mode":"classic","base_score":1,"private":false}`)

	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _ := connectRoomWebSocket(t, server.URL, access.RoomID, token)
	defer conn.Close()

	sendWSJSON(t, conn, map[string]any{
		"type":       "ping",
		"request_id": "req_ping_1",
		"seq":        2,
		"payload": map[string]any{
			"client_time": "2026-06-06T09:00:00Z",
		},
	})

	envelope := readWSEnvelope(t, conn)
	if envelope.Type != "pong" {
		t.Fatalf("type = %q, want %q", envelope.Type, "pong")
	}
	if envelope.RequestID == nil || *envelope.RequestID != "req_ping_1" {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, "req_ping_1")
	}
}

func TestRoomWebSocketUnknownTypeReturnsError(t *testing.T) {
	handler := NewHTTPHandlerWithManager(testConfig(), room.NewManagerWithRNG(&fixedRoomRNG{value: 0}))
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)
	access := createRoomViaAPI(t, handler, token, `{"mode":"classic","base_score":1,"private":false}`)

	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _ := connectRoomWebSocket(t, server.URL, access.RoomID, token)
	defer conn.Close()

	sendWSJSON(t, conn, map[string]any{
		"type":       "room.unknown",
		"request_id": "req_unknown_1",
		"seq":        3,
		"payload":    map[string]any{},
	})

	envelope := readWSEnvelope(t, conn)
	assertWSErrorCode(t, envelope, "req_unknown_1", "unknown_message_type")
}

func TestRoomWebSocketReadyReturnsAckAndUpdatesRoom(t *testing.T) {
	manager := room.NewManagerWithRNG(&fixedRoomRNG{value: 0})
	handler := NewHTTPHandlerWithManager(testConfig(), manager)
	token := loginAndGetToken(t, handler, `{"display_name":"A"}`)
	user := currentUserWithToken(t, handler, token)
	access := createRoomViaAPI(t, handler, token, `{"mode":"classic","base_score":1,"private":false}`)

	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _ := connectRoomWebSocket(t, server.URL, access.RoomID, token)
	defer conn.Close()

	sendWSJSON(t, conn, map[string]any{
		"type":       "room.ready",
		"request_id": "req_ready_1",
		"seq":        4,
		"payload": map[string]any{
			"ready": true,
		},
	})

	envelope := readWSEnvelope(t, conn)
	assertWSAck(t, envelope, "req_ready_1")

	snapshot, err := manager.GetRoomSnapshot(access.RoomID, user.ID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("player len = %d, want 1", len(snapshot.Players))
	}
	if !snapshot.Players[0].Ready {
		t.Fatal("player ready should be true")
	}
}

func TestRoomWebSocketBidReturnsNotPlayerTurn(t *testing.T) {
	setup := newThreePlayerRoom(t)
	readyAllPlayers(t, setup)

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)
	defer conn.Close()

	sendWSJSON(t, conn, map[string]any{
		"type":       "game.bid",
		"request_id": "req_bid_1",
		"seq":        5,
		"payload": map[string]any{
			"score": 1,
		},
	})

	envelope := readWSEnvelope(t, conn)
	assertWSErrorCode(t, envelope, "req_bid_1", "not_player_turn")
}

func TestRoomWebSocketPlayCardsReturnsInvalidCardSet(t *testing.T) {
	setup := newThreePlayerRoom(t)
	readyAllPlayers(t, setup)
	if _, err := setup.Manager.Bid(room.BidInput{
		RoomID: setup.RoomID,
		UserID: setup.Host.ID,
		Score:  3,
	}); err != nil {
		t.Fatalf("Bid() error = %v", err)
	}

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer conn.Close()

	sendWSJSON(t, conn, map[string]any{
		"type":       "game.play_cards",
		"request_id": "req_play_1",
		"seq":        6,
		"payload": map[string]any{
			"cards": []string{"ZZ"},
		},
	})

	envelope := readWSEnvelope(t, conn)
	assertWSErrorCode(t, envelope, "req_play_1", "invalid_card_set")
}

func TestRoomWebSocketPassReturnsCannotPass(t *testing.T) {
	setup := newThreePlayerRoom(t)
	readyAllPlayers(t, setup)
	if _, err := setup.Manager.Bid(room.BidInput{
		RoomID: setup.RoomID,
		UserID: setup.Host.ID,
		Score:  3,
	}); err != nil {
		t.Fatalf("Bid() error = %v", err)
	}

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer conn.Close()

	sendWSJSON(t, conn, map[string]any{
		"type":       "game.pass",
		"request_id": "req_pass_1",
		"seq":        7,
		"payload":    map[string]any{},
	})

	envelope := readWSEnvelope(t, conn)
	assertWSErrorCode(t, envelope, "req_pass_1", "cannot_pass")
}

func createRoomViaAPI(t *testing.T, handler http.Handler, token string, payload string) roomAccessData {
	t.Helper()

	rec := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms", token, payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("create room status = %d, want %d", rec.Code, http.StatusOK)
	}

	var data roomAccessData
	decodeResponseData(t, decodeResponseEnvelope(t, rec), &data)
	return data
}

func currentUserWithToken(t *testing.T, handler http.Handler, token string) currentUserData {
	t.Helper()

	rec := doAuthenticatedJSONRequest(t, handler, http.MethodGet, "/api/v1/auth/me", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("current user status = %d, want %d", rec.Code, http.StatusOK)
	}

	var data currentUserData
	decodeResponseData(t, decodeResponseEnvelope(t, rec), &data)
	return data
}

func newThreePlayerRoom(t *testing.T) wsTestRoom {
	t.Helper()

	manager := room.NewManagerWithRNG(&fixedRoomRNG{value: 0})
	handler := NewHTTPHandlerWithManager(testConfig(), manager)

	hostToken := loginAndGetToken(t, handler, `{"display_name":"Host"}`)
	user2Token := loginAndGetToken(t, handler, `{"display_name":"User2"}`)
	user3Token := loginAndGetToken(t, handler, `{"display_name":"User3"}`)

	host := currentUserWithToken(t, handler, hostToken)
	user2 := currentUserWithToken(t, handler, user2Token)
	user3 := currentUserWithToken(t, handler, user3Token)

	access := createRoomViaAPI(t, handler, hostToken, `{"mode":"classic","base_score":1,"private":false}`)

	join2 := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms/"+access.RoomID+"/join", user2Token, `{"preferred_seat":1}`)
	if join2.Code != http.StatusOK {
		t.Fatalf("join user2 status = %d, want %d", join2.Code, http.StatusOK)
	}

	join3 := doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms/"+access.RoomID+"/join", user3Token, `{"preferred_seat":2}`)
	if join3.Code != http.StatusOK {
		t.Fatalf("join user3 status = %d, want %d", join3.Code, http.StatusOK)
	}

	return wsTestRoom{
		Manager: manager,
		Handler: handler,
		RoomID:  access.RoomID,
		Host: wsTestUser{
			ID:    host.ID,
			Token: hostToken,
		},
		User2: wsTestUser{
			ID:    user2.ID,
			Token: user2Token,
		},
		User3: wsTestUser{
			ID:    user3.ID,
			Token: user3Token,
		},
	}
}

func readyAllPlayers(t *testing.T, setup wsTestRoom) {
	t.Helper()

	users := []wsTestUser{setup.Host, setup.User2, setup.User3}
	for _, user := range users {
		if _, _, _, err := setup.Manager.Ready(room.ReadyInput{
			RoomID: setup.RoomID,
			UserID: user.ID,
			Ready:  true,
		}); err != nil {
			t.Fatalf("Ready(%s) error = %v", user.ID, err)
		}
	}
}

func connectRoomWebSocket(t *testing.T, serverURL string, roomID string, token string) (*websocket.Conn, room.RoomSnapshot) {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(buildRoomWSConnectURL(serverURL, roomID, token), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}

	envelope := readWSEnvelope(t, conn)
	if envelope.Type != "room.snapshot" {
		t.Fatalf("initial type = %q, want %q", envelope.Type, "room.snapshot")
	}

	var snapshot room.RoomSnapshot
	if err := json.Unmarshal(envelope.Payload, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}

	return conn, snapshot
}

func sendWSJSON(t *testing.T, conn *websocket.Conn, payload any) {
	t.Helper()
	if err := conn.WriteJSON(payload); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
}

func readWSEnvelope(t *testing.T, conn *websocket.Conn) wsServerEnvelope {
	t.Helper()

	var envelope wsServerEnvelope
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	return envelope
}

func assertWSAck(t *testing.T, envelope wsServerEnvelope, requestID string) {
	t.Helper()

	if envelope.Type != "ack" {
		t.Fatalf("type = %q, want %q", envelope.Type, "ack")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsAckPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal ack payload: %v", err)
	}
	if !payload.Ok {
		t.Fatal("ack payload ok should be true")
	}
}

func assertWSErrorCode(t *testing.T, envelope wsServerEnvelope, requestID string, code string) {
	t.Helper()

	if envelope.Type != "error" {
		t.Fatalf("type = %q, want %q", envelope.Type, "error")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsErrorPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal error payload: %v", err)
	}
	if payload.Code != code {
		t.Fatalf("error code = %q, want %q", payload.Code, code)
	}
}

func buildRoomWSConnectURL(serverURL string, roomID string, token string) string {
	wsURL := buildRoomWSBaseURL(serverURL, roomID)
	if token == "" {
		return wsURL
	}
	return wsURL + "?token=" + url.QueryEscape(token)
}

func buildRoomWSBaseURL(serverURL string, roomID string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + "/ws/v1/rooms/" + roomID
}
