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
