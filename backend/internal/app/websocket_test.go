package app

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"ddz/backend/internal/game"
	"ddz/backend/internal/record"
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

type wsRoomPlayerReadyPayload struct {
	UserID    string `json:"user_id"`
	SeatIndex int    `json:"seat_index"`
	Ready     bool   `json:"ready"`
}

type wsGameBidPlacedPayload struct {
	UserID        string    `json:"user_id"`
	SeatIndex     int       `json:"seat_index"`
	Score         int       `json:"score"`
	NextSeatIndex int       `json:"next_seat_index"`
	DeadlineAt    time.Time `json:"deadline_at"`
}

type wsGameLandlordDecidedPayload struct {
	LandlordSeatIndex int       `json:"landlord_seat_index"`
	LandlordUserID    string    `json:"landlord_user_id"`
	BottomCards       []string  `json:"bottom_cards"`
	Multiplier        int       `json:"multiplier"`
	CurrentSeatIndex  int       `json:"current_seat_index"`
	DeadlineAt        time.Time `json:"deadline_at"`
}

type wsGameCardsPlayedPayload struct {
	UserID    string   `json:"user_id"`
	SeatIndex int      `json:"seat_index"`
	Cards     []string `json:"cards"`
	CardGroup struct {
		Type   string `json:"type"`
		Rank   string `json:"rank"`
		Length int    `json:"length"`
	} `json:"card_group"`
	RemainingCount int       `json:"remaining_count"`
	NextSeatIndex  int       `json:"next_seat_index"`
	DeadlineAt     time.Time `json:"deadline_at"`
}

type wsGameMyHandUpdatedPayload struct {
	Cards []string `json:"cards"`
}

type wsGamePlayerPassedPayload struct {
	UserID        string    `json:"user_id"`
	SeatIndex     int       `json:"seat_index"`
	NextSeatIndex int       `json:"next_seat_index"`
	DeadlineAt    time.Time `json:"deadline_at"`
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

func TestRoomWebSocketReadyBroadcastsPublicEvent(t *testing.T) {
	setup := newThreePlayerRoom(t)

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	hostConn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer hostConn.Close()
	user2Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)
	defer user2Conn.Close()

	sendWSJSON(t, hostConn, map[string]any{
		"type":       "room.ready",
		"request_id": "req_ready_broadcast_1",
		"seq":        8,
		"payload": map[string]any{
			"ready": true,
		},
	})

	assertWSAck(t, readWSEnvelope(t, hostConn), "req_ready_broadcast_1")

	hostEvent := readWSEnvelope(t, hostConn)
	assertRoomPlayerReadyEvent(t, hostEvent, "req_ready_broadcast_1", setup.Host.ID, 0, true)

	user2Event := readWSEnvelope(t, user2Conn)
	assertRoomPlayerReadyEvent(t, user2Event, "req_ready_broadcast_1", setup.Host.ID, 0, true)
}

func TestRoomWebSocketReadyStartsGameAndSendsPrivateSnapshots(t *testing.T) {
	setup := newThreePlayerRoom(t)
	if _, _, _, err := setup.Manager.Ready(room.ReadyInput{RoomID: setup.RoomID, UserID: setup.Host.ID, Ready: true}); err != nil {
		t.Fatalf("Ready host error = %v", err)
	}
	if _, _, _, err := setup.Manager.Ready(room.ReadyInput{RoomID: setup.RoomID, UserID: setup.User2.ID, Ready: true}); err != nil {
		t.Fatalf("Ready user2 error = %v", err)
	}

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	hostConn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer hostConn.Close()
	user2Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)
	defer user2Conn.Close()
	user3Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User3.Token)
	defer user3Conn.Close()

	sendWSJSON(t, user3Conn, map[string]any{
		"type":       "room.ready",
		"request_id": "req_ready_start_1",
		"seq":        9,
		"payload": map[string]any{
			"ready": true,
		},
	})

	assertWSAck(t, readWSEnvelope(t, user3Conn), "req_ready_start_1")
	assertRoomPlayerReadyEvent(t, readWSEnvelope(t, hostConn), "req_ready_start_1", setup.User3.ID, 2, true)
	assertRoomPlayerReadyEvent(t, readWSEnvelope(t, user2Conn), "req_ready_start_1", setup.User3.ID, 2, true)
	assertRoomPlayerReadyEvent(t, readWSEnvelope(t, user3Conn), "req_ready_start_1", setup.User3.ID, 2, true)

	hostSnapshot := readRoomSnapshotEvent(t, hostConn)
	user2Snapshot := readRoomSnapshotEvent(t, user2Conn)
	user3Snapshot := readRoomSnapshotEvent(t, user3Conn)

	assertStartedSnapshot(t, hostSnapshot, 17)
	assertStartedSnapshot(t, user2Snapshot, 17)
	assertStartedSnapshot(t, user3Snapshot, 17)
}

func TestRoomWebSocketBidBroadcastsLandlordDecidedAndPrivateHandUpdate(t *testing.T) {
	setup := newThreePlayerRoom(t)
	readyAllPlayers(t, setup)

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	hostConn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer hostConn.Close()
	user2Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)
	defer user2Conn.Close()

	sendWSJSON(t, hostConn, map[string]any{
		"type":       "game.bid",
		"request_id": "req_bid_broadcast_1",
		"seq":        10,
		"payload": map[string]any{
			"score": 3,
		},
	})

	assertWSAck(t, readWSEnvelope(t, hostConn), "req_bid_broadcast_1")

	hostBidEvent := readWSEnvelope(t, hostConn)
	user2BidEvent := readWSEnvelope(t, user2Conn)
	assertBidPlacedEvent(t, hostBidEvent, "req_bid_broadcast_1", setup.Host.ID, 0, 3)
	assertBidPlacedEvent(t, user2BidEvent, "req_bid_broadcast_1", setup.Host.ID, 0, 3)

	hostLandlordEvent := readWSEnvelope(t, hostConn)
	user2LandlordEvent := readWSEnvelope(t, user2Conn)
	assertLandlordDecidedEvent(t, hostLandlordEvent, "req_bid_broadcast_1", setup.Host.ID, 0, 3)
	assertLandlordDecidedEvent(t, user2LandlordEvent, "req_bid_broadcast_1", setup.Host.ID, 0, 3)

	hostHandEvent := readWSEnvelope(t, hostConn)
	assertHandUpdatedEvent(t, hostHandEvent, "req_bid_broadcast_1", 20)
	assertNoWSMessage(t, user2Conn)
}

func TestRoomWebSocketPlayBroadcastsCardsPlayedAndPrivateHandUpdate(t *testing.T) {
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

	hostConn, hostSnapshot := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer hostConn.Close()
	user2Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)
	defer user2Conn.Close()

	if len(hostSnapshot.Me.Hand) == 0 {
		t.Fatal("host hand should not be empty")
	}
	playCard := hostSnapshot.Me.Hand[0]

	sendWSJSON(t, hostConn, map[string]any{
		"type":       "game.play_cards",
		"request_id": "req_play_broadcast_1",
		"seq":        11,
		"payload": map[string]any{
			"cards": []string{playCard},
		},
	})

	assertWSAck(t, readWSEnvelope(t, hostConn), "req_play_broadcast_1")

	hostPlayedEvent := readWSEnvelope(t, hostConn)
	user2PlayedEvent := readWSEnvelope(t, user2Conn)
	assertCardsPlayedEvent(t, hostPlayedEvent, "req_play_broadcast_1", setup.Host.ID, 0, playCard, 19)
	assertCardsPlayedEvent(t, user2PlayedEvent, "req_play_broadcast_1", setup.Host.ID, 0, playCard, 19)

	hostHandEvent := readWSEnvelope(t, hostConn)
	assertHandUpdatedEvent(t, hostHandEvent, "req_play_broadcast_1", 19)
	assertNoWSMessage(t, user2Conn)
}

func TestRoomWebSocketPassBroadcastsPublicEvent(t *testing.T) {
	setup := newThreePlayerRoom(t)
	readyAllPlayers(t, setup)
	if _, err := setup.Manager.Bid(room.BidInput{
		RoomID: setup.RoomID,
		UserID: setup.Host.ID,
		Score:  3,
	}); err != nil {
		t.Fatalf("Bid() error = %v", err)
	}

	hostSnapshot, err := setup.Manager.GetRoomSnapshot(setup.RoomID, setup.Host.ID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}
	firstCard, err := game.ParseCard(hostSnapshot.Me.Hand[0])
	if err != nil {
		t.Fatalf("ParseCard() error = %v", err)
	}
	if _, err := setup.Manager.PlayCards(room.PlayCardsInput{
		RoomID: setup.RoomID,
		UserID: setup.Host.ID,
		Cards:  []game.Card{firstCard},
	}); err != nil {
		t.Fatalf("PlayCards() error = %v", err)
	}

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	user2Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)
	defer user2Conn.Close()
	user3Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User3.Token)
	defer user3Conn.Close()

	sendWSJSON(t, user2Conn, map[string]any{
		"type":       "game.pass",
		"request_id": "req_pass_broadcast_1",
		"seq":        12,
		"payload":    map[string]any{},
	})

	assertWSAck(t, readWSEnvelope(t, user2Conn), "req_pass_broadcast_1")

	user2PassEvent := readWSEnvelope(t, user2Conn)
	user3PassEvent := readWSEnvelope(t, user3Conn)
	assertPlayerPassedEvent(t, user2PassEvent, "req_pass_broadcast_1", setup.User2.ID, 1)
	assertPlayerPassedEvent(t, user3PassEvent, "req_pass_broadcast_1", setup.User2.ID, 1)
	assertNoWSMessage(t, user3Conn)
}

func TestRoomWebSocketDisconnectMarksPlayerOffline(t *testing.T) {
	setup := newThreePlayerRoom(t)
	readyAllPlayers(t, setup)

	server := httptest.NewServer(setup.Handler)
	defer server.Close()

	hostConn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer hostConn.Close()
	user2Conn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)

	if err := user2Conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	var snapshot room.RoomSnapshot
	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("did not receive offline snapshot in time")
		}

		envelope := readWSEnvelope(t, hostConn)
		if envelope.Type != "room.snapshot" {
			continue
		}
		if err := json.Unmarshal(envelope.Payload, &snapshot); err != nil {
			t.Fatalf("unmarshal snapshot: %v", err)
		}
		if findPlayerStatus(snapshot.Players, setup.User2.ID) == "offline" {
			break
		}
	}

	if findPlayerStatus(snapshot.Players, setup.User2.ID) != "offline" {
		t.Fatalf("player status = %q, want %q", findPlayerStatus(snapshot.Players, setup.User2.ID), "offline")
	}
}

func TestRoomWebSocketReconnectRestoresPrivateSnapshotAndOnlineStatus(t *testing.T) {
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

	hostConn, _ := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.Host.Token)
	defer hostConn.Close()
	user2Conn, firstSnapshot := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)

	if len(firstSnapshot.Me.Hand) == 0 {
		t.Fatal("initial hand should not be empty")
	}

	if err := user2Conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	waitForPlayerStatusSnapshot(t, hostConn, setup.User2.ID, "offline")

	reconnectConn, reconnectSnapshot := connectRoomWebSocket(t, server.URL, setup.RoomID, setup.User2.Token)
	defer reconnectConn.Close()

	if findPlayerStatus(reconnectSnapshot.Players, setup.User2.ID) != "playing" {
		t.Fatalf("reconnect status = %q, want %q", findPlayerStatus(reconnectSnapshot.Players, setup.User2.ID), "playing")
	}
	if len(reconnectSnapshot.Me.Hand) == 0 {
		t.Fatal("reconnect hand should not be empty")
	}
	if len(reconnectSnapshot.Me.Hand) != len(firstSnapshot.Me.Hand) {
		t.Fatalf("reconnect hand len = %d, want %d", len(reconnectSnapshot.Me.Hand), len(firstSnapshot.Me.Hand))
	}

	waitForPlayerStatusSnapshot(t, hostConn, setup.User2.ID, "playing")
}

func TestCompletedGameCanBeQueriedFromRecordAPI(t *testing.T) {
	manager := room.NewManagerWithRNG(&fixedRoomRNG{value: 0})
	recordService := record.NewService(record.NewMemoryStore())
	handler := NewHTTPHandlerWithDependencies(testConfig(), manager, recordService)

	hostToken := loginAndGetToken(t, handler, `{"display_name":"Host"}`)
	user2Token := loginAndGetToken(t, handler, `{"display_name":"User2"}`)
	user3Token := loginAndGetToken(t, handler, `{"display_name":"User3"}`)

	host := currentUserWithToken(t, handler, hostToken)
	user2 := currentUserWithToken(t, handler, user2Token)
	user3 := currentUserWithToken(t, handler, user3Token)

	access := createRoomViaAPI(t, handler, hostToken, `{"mode":"classic","base_score":1,"private":false}`)
	_ = doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms/"+access.RoomID+"/join", user2Token, `{"preferred_seat":1}`)
	_ = doAuthenticatedJSONRequest(t, handler, http.MethodPost, "/api/v1/rooms/"+access.RoomID+"/join", user3Token, `{"preferred_seat":2}`)

	users := []string{host.ID, user2.ID, user3.ID}
	for _, userID := range users {
		if _, _, _, err := manager.Ready(room.ReadyInput{RoomID: access.RoomID, UserID: userID, Ready: true}); err != nil {
			t.Fatalf("Ready(%s) error = %v", userID, err)
		}
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	hostConn, _ := connectRoomWebSocket(t, server.URL, access.RoomID, hostToken)
	defer hostConn.Close()

	sendWSJSON(t, hostConn, map[string]any{
		"type":       "game.bid",
		"request_id": "req_record_bid",
		"seq":        1,
		"payload": map[string]any{
			"score": 3,
		},
	})
	assertWSAck(t, readWSEnvelope(t, hostConn), "req_record_bid")
	_ = readWSEnvelope(t, hostConn)
	_ = readWSEnvelope(t, hostConn)
	_ = readWSEnvelope(t, hostConn)

	currentRoom, err := manager.GetRoom(access.RoomID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if currentRoom.CurrentGame == nil || len(currentRoom.CurrentGame.Players[0].Hand) == 0 {
		t.Fatal("host game hand should not be empty")
	}
	winningCard := currentRoom.CurrentGame.Players[0].Hand[0]
	currentRoom.CurrentGame.Players[0].Hand = []game.Card{winningCard}
	currentRoom.CurrentGame.Players[0].RemainingCount = 1
	currentRoom.CurrentGame.CurrentSeatIndex = 0
	currentRoom.CurrentGame.LastPlay = nil

	sendWSJSON(t, hostConn, map[string]any{
		"type":       "game.play_cards",
		"request_id": "req_record_play",
		"seq":        2,
		"payload": map[string]any{
			"cards": []string{winningCard.Code()},
		},
	})

	assertWSAck(t, readWSEnvelope(t, hostConn), "req_record_play")
	deadline := time.Now().Add(2 * time.Second)
	var ended bool
	for time.Now().Before(deadline) {
		envelope := readWSEnvelope(t, hostConn)
		if envelope.Type == "game.ended" {
			ended = true
			break
		}
	}
	if !ended {
		t.Fatal("did not receive game.ended event")
	}

	listRec := doAuthenticatedJSONRequest(t, handler, http.MethodGet, "/api/v1/records/my?page=1&page_size=20", hostToken, "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	var listData recordListData
	decodeResponseData(t, decodeResponseEnvelope(t, listRec), &listData)
	if listData.Total != 1 {
		t.Fatalf("total = %d, want 1", listData.Total)
	}

	detailRec := doAuthenticatedJSONRequest(t, handler, http.MethodGet, "/api/v1/records/g_"+access.RoomID, hostToken, "")
	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", detailRec.Code, http.StatusOK)
	}
	var detailData recordDetailData
	decodeResponseData(t, decodeResponseEnvelope(t, detailRec), &detailData)
	if detailData.GameID != "g_"+access.RoomID {
		t.Fatalf("game_id = %q, want %q", detailData.GameID, "g_"+access.RoomID)
	}
	if len(detailData.Events) < 2 {
		t.Fatalf("events len = %d, want >= 2", len(detailData.Events))
	}

	hostMe := currentUserWithToken(t, handler, hostToken)
	if hostMe.Profile.TotalGames != 1 {
		t.Fatalf("host total_games = %d, want 1", hostMe.Profile.TotalGames)
	}
	if hostMe.Profile.Wins != 1 {
		t.Fatalf("host wins = %d, want 1", hostMe.Profile.Wins)
	}

	user2Me := currentUserWithToken(t, handler, user2Token)
	if user2Me.Profile.TotalGames != 1 {
		t.Fatalf("user2 total_games = %d, want 1", user2Me.Profile.TotalGames)
	}
	if user2Me.Profile.Wins != 0 {
		t.Fatalf("user2 wins = %d, want 0", user2Me.Profile.Wins)
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

func readRoomSnapshotEvent(t *testing.T, conn *websocket.Conn) room.RoomSnapshot {
	t.Helper()

	envelope := readWSEnvelope(t, conn)
	if envelope.Type != "room.snapshot" {
		t.Fatalf("type = %q, want %q", envelope.Type, "room.snapshot")
	}

	var snapshot room.RoomSnapshot
	if err := json.Unmarshal(envelope.Payload, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	return snapshot
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

func assertRoomPlayerReadyEvent(t *testing.T, envelope wsServerEnvelope, requestID string, userID string, seatIndex int, ready bool) {
	t.Helper()

	if envelope.Type != "room.player_ready" {
		t.Fatalf("type = %q, want %q", envelope.Type, "room.player_ready")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsRoomPlayerReadyPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal room.player_ready payload: %v", err)
	}
	if payload.UserID != userID || payload.SeatIndex != seatIndex || payload.Ready != ready {
		t.Fatalf("payload = %+v, want user_id=%q seat_index=%d ready=%v", payload, userID, seatIndex, ready)
	}
}

func assertStartedSnapshot(t *testing.T, snapshot room.RoomSnapshot, wantHandLen int) {
	t.Helper()

	if snapshot.Game == nil {
		t.Fatal("snapshot game should not be nil")
	}
	if snapshot.Game.Phase != "bidding" {
		t.Fatalf("game phase = %q, want %q", snapshot.Game.Phase, "bidding")
	}
	if len(snapshot.Me.Hand) != wantHandLen {
		t.Fatalf("me hand len = %d, want %d", len(snapshot.Me.Hand), wantHandLen)
	}
}

func assertBidPlacedEvent(t *testing.T, envelope wsServerEnvelope, requestID string, userID string, seatIndex int, score int) {
	t.Helper()

	if envelope.Type != "game.bid_placed" {
		t.Fatalf("type = %q, want %q", envelope.Type, "game.bid_placed")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsGameBidPlacedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal game.bid_placed payload: %v", err)
	}
	if payload.UserID != userID || payload.SeatIndex != seatIndex || payload.Score != score {
		t.Fatalf("payload = %+v, want user_id=%q seat_index=%d score=%d", payload, userID, seatIndex, score)
	}
}

func assertLandlordDecidedEvent(t *testing.T, envelope wsServerEnvelope, requestID string, landlordUserID string, landlordSeatIndex int, multiplier int) {
	t.Helper()

	if envelope.Type != "game.landlord_decided" {
		t.Fatalf("type = %q, want %q", envelope.Type, "game.landlord_decided")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsGameLandlordDecidedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal game.landlord_decided payload: %v", err)
	}
	if payload.LandlordUserID != landlordUserID || payload.LandlordSeatIndex != landlordSeatIndex || payload.Multiplier != multiplier {
		t.Fatalf("payload = %+v, want landlord_user_id=%q landlord_seat_index=%d multiplier=%d", payload, landlordUserID, landlordSeatIndex, multiplier)
	}
	if len(payload.BottomCards) != 3 {
		t.Fatalf("bottom_cards len = %d, want 3", len(payload.BottomCards))
	}
}

func assertHandUpdatedEvent(t *testing.T, envelope wsServerEnvelope, requestID string, wantHandLen int) {
	t.Helper()

	if envelope.Type != "game.my_hand_updated" {
		t.Fatalf("type = %q, want %q", envelope.Type, "game.my_hand_updated")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsGameMyHandUpdatedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal game.my_hand_updated payload: %v", err)
	}
	if len(payload.Cards) != wantHandLen {
		t.Fatalf("hand len = %d, want %d", len(payload.Cards), wantHandLen)
	}
}

func assertCardsPlayedEvent(t *testing.T, envelope wsServerEnvelope, requestID string, userID string, seatIndex int, card string, remainingCount int) {
	t.Helper()

	if envelope.Type != "game.cards_played" {
		t.Fatalf("type = %q, want %q", envelope.Type, "game.cards_played")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsGameCardsPlayedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal game.cards_played payload: %v", err)
	}
	if payload.UserID != userID || payload.SeatIndex != seatIndex {
		t.Fatalf("payload = %+v, want user_id=%q seat_index=%d", payload, userID, seatIndex)
	}
	if len(payload.Cards) != 1 || payload.Cards[0] != card {
		t.Fatalf("cards = %+v, want [%q]", payload.Cards, card)
	}
	if payload.RemainingCount != remainingCount {
		t.Fatalf("remaining_count = %d, want %d", payload.RemainingCount, remainingCount)
	}
}

func assertPlayerPassedEvent(t *testing.T, envelope wsServerEnvelope, requestID string, userID string, seatIndex int) {
	t.Helper()

	if envelope.Type != "game.player_passed" {
		t.Fatalf("type = %q, want %q", envelope.Type, "game.player_passed")
	}
	if envelope.RequestID == nil || *envelope.RequestID != requestID {
		t.Fatalf("request_id = %v, want %q", envelope.RequestID, requestID)
	}

	var payload wsGamePlayerPassedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal game.player_passed payload: %v", err)
	}
	if payload.UserID != userID || payload.SeatIndex != seatIndex {
		t.Fatalf("payload = %+v, want user_id=%q seat_index=%d", payload, userID, seatIndex)
	}
}

func assertNoWSMessage(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(120 * time.Millisecond))
	defer conn.SetReadDeadline(time.Time{})

	var envelope wsServerEnvelope
	err := conn.ReadJSON(&envelope)
	if err == nil {
		t.Fatalf("unexpected websocket message: %+v", envelope)
	}

	netErr, ok := err.(net.Error)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected timeout, got %v", err)
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

func findPlayerStatus(players []room.RoomSnapshotPlayer, userID string) string {
	for _, player := range players {
		if player.UserID == userID {
			return player.Status
		}
	}
	return ""
}

func waitForPlayerStatusSnapshot(t *testing.T, conn *websocket.Conn, userID string, wantStatus string) room.RoomSnapshot {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("did not receive %q snapshot in time", wantStatus)
		}

		envelope := readWSEnvelope(t, conn)
		if envelope.Type != "room.snapshot" {
			continue
		}

		var snapshot room.RoomSnapshot
		if err := json.Unmarshal(envelope.Payload, &snapshot); err != nil {
			t.Fatalf("unmarshal snapshot: %v", err)
		}
		if findPlayerStatus(snapshot.Players, userID) == wantStatus {
			return snapshot
		}
	}
}
