package ws

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ddz/backend/internal/auth"
	"ddz/backend/internal/game"
	"ddz/backend/internal/room"

	"github.com/gorilla/websocket"
)

const (
	defaultReadLimit      int64 = 64 << 10
	defaultReadWait             = 45 * time.Second
	defaultWriteWait            = 10 * time.Second
	defaultPingPeriod           = 30 * time.Second
	defaultSendBufferSize       = 16
	roomRoutePrefix             = "/ws/v1/rooms/"
)

var errConnectionClosed = errors.New("websocket connection closed")
var errSendBufferFull = errors.New("websocket send buffer full")

type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id,omitempty"`
}

type serverMessage struct {
	Type       string    `json:"type"`
	RequestID  *string   `json:"request_id"`
	Seq        uint64    `json:"seq"`
	ServerTime time.Time `json:"server_time"`
	Payload    any       `json:"payload"`
}

type clientMessage struct {
	Type      string          `json:"type"`
	RequestID *string         `json:"request_id"`
	Seq       uint64          `json:"seq"`
	Payload   json.RawMessage `json:"payload"`
}

type ackPayload struct {
	Ok bool `json:"ok"`
}

type messageErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type pingPayload struct {
	ClientTime string `json:"client_time"`
}

type readyPayload struct {
	Ready bool `json:"ready"`
}

type bidPayload struct {
	Score int `json:"score"`
}

type playCardsPayload struct {
	Cards []string `json:"cards"`
}

// Gateway 负责房间 WebSocket 连接的升级、鉴权与生命周期管理。
type Gateway struct {
	jwt         *auth.JWTManager
	roomManager *room.Manager
	upgrader    websocket.Upgrader
	now         func() time.Time

	mu    sync.Mutex
	rooms map[string]map[string]*clientConn
}

type clientConn struct {
	gateway *Gateway
	roomID  string
	userID  string
	conn    *websocket.Conn
	send    chan []byte
	done    chan struct{}
	seq     atomic.Uint64

	closeOnce sync.Once
}

// NewGateway 创建房间 WebSocket 网关。
func NewGateway(jwt *auth.JWTManager, roomManager *room.Manager) *Gateway {
	return &Gateway{
		jwt:         jwt,
		roomManager: roomManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		now: func() time.Time {
			return time.Now().UTC()
		},
		rooms: make(map[string]map[string]*clientConn),
	}
}

// ServeHTTP 处理房间 WebSocket 连接请求。
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}
	if g == nil || g.jwt == nil || g.roomManager == nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	roomID, ok := parseRoomPath(r.URL.Path)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found", "not found")
		return
	}

	token, ok := extractAccessToken(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	claims, err := g.jwt.Parse(token)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	snapshot, err := g.roomManager.GetRoomSnapshot(roomID, claims.Subject)
	if err != nil {
		statusCode, code, message := mapRoomError(err)
		writeError(w, r, statusCode, code, message)
		return
	}

	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &clientConn{
		gateway: g,
		roomID:  roomID,
		userID:  claims.Subject,
		conn:    conn,
		send:    make(chan []byte, defaultSendBufferSize),
		done:    make(chan struct{}),
	}

	g.register(client)
	go client.writeLoop()

	if err := client.sendMessage("room.snapshot", nil, snapshot); err != nil {
		g.unregister(client)
		return
	}

	client.readLoop()
}

func (g *Gateway) register(client *clientConn) {
	g.mu.Lock()
	roomClients := g.rooms[client.roomID]
	if roomClients == nil {
		roomClients = make(map[string]*clientConn)
		g.rooms[client.roomID] = roomClients
	}
	previous := roomClients[client.userID]
	roomClients[client.userID] = client
	g.mu.Unlock()

	if previous != nil && previous != client {
		previous.close()
	}
}

func (g *Gateway) unregister(client *clientConn) {
	g.mu.Lock()
	if roomClients, ok := g.rooms[client.roomID]; ok {
		if current, exists := roomClients[client.userID]; exists && current == client {
			delete(roomClients, client.userID)
			if len(roomClients) == 0 {
				delete(g.rooms, client.roomID)
			}
		}
	}
	g.mu.Unlock()

	client.close()
}

func (c *clientConn) readLoop() {
	defer c.gateway.unregister(c)

	c.conn.SetReadLimit(defaultReadLimit)
	_ = c.conn.SetReadDeadline(c.gateway.now().Add(defaultReadWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(c.gateway.now().Add(defaultReadWait))
	})
	c.conn.SetPingHandler(func(appData string) error {
		if err := c.conn.SetReadDeadline(c.gateway.now().Add(defaultReadWait)); err != nil {
			return err
		}
		return c.conn.WriteControl(websocket.PongMessage, []byte(appData), c.gateway.now().Add(defaultWriteWait))
	})

	for {
		messageType, reader, err := c.conn.NextReader()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}

		payload, err := io.ReadAll(reader)
		if err != nil {
			return
		}

		if messageType != websocket.TextMessage {
			if err := c.sendError(nil, "bad_request", "bad request"); err != nil {
				return
			}
			continue
		}
		if err := c.handleMessage(payload); err != nil {
			return
		}

		_ = c.conn.SetReadDeadline(c.gateway.now().Add(defaultReadWait))
	}
}

func (c *clientConn) writeLoop() {
	ticker := time.NewTicker(defaultPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case payload := <-c.send:
			if err := c.writeText(payload); err != nil {
				c.gateway.unregister(c)
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, c.gateway.now().Add(defaultWriteWait)); err != nil {
				c.gateway.unregister(c)
				return
			}
		}
	}
}

func (c *clientConn) writeText(payload []byte) error {
	if err := c.conn.SetWriteDeadline(c.gateway.now().Add(defaultWriteWait)); err != nil {
		return err
	}

	writer, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func (c *clientConn) sendMessage(messageType string, requestID *string, payload any) error {
	envelope := serverMessage{
		Type:       messageType,
		RequestID:  requestID,
		Seq:        c.seq.Add(1),
		ServerTime: c.gateway.now(),
		Payload:    payload,
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	select {
	case <-c.done:
		return errConnectionClosed
	case c.send <- data:
		return nil
	default:
		return errSendBufferFull
	}
}

func (c *clientConn) handleMessage(raw []byte) error {
	var message clientMessage
	if err := decodeJSON(raw, &message); err != nil {
		return c.sendError(nil, "bad_request", "bad request")
	}

	switch message.Type {
	case "ping":
		return c.handlePing(message)
	case "room.ready":
		return c.handleReady(message)
	case "game.bid":
		return c.handleBid(message)
	case "game.play_cards":
		return c.handlePlayCards(message)
	case "game.pass":
		return c.handlePass(message)
	case "":
		return c.sendError(message.RequestID, "bad_request", "bad request")
	default:
		return c.sendError(message.RequestID, "unknown_message_type", "unknown message type")
	}
}

func (c *clientConn) handlePing(message clientMessage) error {
	var payload pingPayload
	if err := decodePayload(message.Payload, &payload); err != nil {
		return c.sendError(message.RequestID, "bad_request", "bad request")
	}
	return c.sendMessage("pong", message.RequestID, struct{}{})
}

func (c *clientConn) handleReady(message clientMessage) error {
	var payload readyPayload
	if err := decodePayload(message.Payload, &payload); err != nil {
		return c.sendError(message.RequestID, "bad_request", "bad request")
	}

	if _, _, _, err := c.gateway.roomManager.Ready(room.ReadyInput{
		RoomID: c.roomID,
		UserID: c.userID,
		Ready:  payload.Ready,
	}); err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	return c.sendAck(message.RequestID)
}

func (c *clientConn) handleBid(message clientMessage) error {
	var payload bidPayload
	if err := decodePayload(message.Payload, &payload); err != nil {
		return c.sendError(message.RequestID, "bad_request", "bad request")
	}

	if _, err := c.gateway.roomManager.Bid(room.BidInput{
		RoomID: c.roomID,
		UserID: c.userID,
		Score:  payload.Score,
	}); err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	return c.sendAck(message.RequestID)
}

func (c *clientConn) handlePlayCards(message clientMessage) error {
	var payload playCardsPayload
	if err := decodePayload(message.Payload, &payload); err != nil {
		return c.sendError(message.RequestID, "bad_request", "bad request")
	}

	cards, err := parseCardCodes(payload.Cards)
	if err != nil {
		return c.sendError(message.RequestID, "invalid_card_set", "invalid card set")
	}

	if _, err := c.gateway.roomManager.PlayCards(room.PlayCardsInput{
		RoomID: c.roomID,
		UserID: c.userID,
		Cards:  cards,
	}); err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	return c.sendAck(message.RequestID)
}

func (c *clientConn) handlePass(message clientMessage) error {
	var payload struct{}
	if err := decodePayload(message.Payload, &payload); err != nil {
		return c.sendError(message.RequestID, "bad_request", "bad request")
	}

	if _, err := c.gateway.roomManager.Pass(room.PassInput{
		RoomID: c.roomID,
		UserID: c.userID,
	}); err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	return c.sendAck(message.RequestID)
}

func (c *clientConn) sendAck(requestID *string) error {
	return c.sendMessage("ack", requestID, ackPayload{Ok: true})
}

func (c *clientConn) sendActionError(requestID *string, err error) error {
	code, message := mapActionError(err)
	return c.sendError(requestID, code, message)
}

func (c *clientConn) sendError(requestID *string, code string, message string) error {
	return c.sendMessage("error", requestID, messageErrorPayload{
		Code:    code,
		Message: message,
	})
}

func (c *clientConn) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func parseRoomPath(path string) (string, bool) {
	if !strings.HasPrefix(path, roomRoutePrefix) {
		return "", false
	}

	roomID := strings.Trim(strings.TrimPrefix(path, roomRoutePrefix), "/")
	if roomID == "" || strings.Contains(roomID, "/") {
		return "", false
	}
	return roomID, true
}

func extractAccessToken(r *http.Request) (string, bool) {
	if token, ok := extractBearerToken(r.Header.Get("Authorization")); ok {
		return token, true
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		return "", false
	}
	return token, true
}

func extractBearerToken(header string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func mapRoomError(err error) (int, string, string) {
	switch {
	case errors.Is(err, room.ErrRoomNotFound), errors.Is(err, room.ErrRoomClosed):
		return http.StatusNotFound, "room_not_found", "room not found"
	case errors.Is(err, room.ErrUserNotInRoom):
		return http.StatusForbidden, "not_in_room", "not in room"
	case errors.Is(err, room.ErrInvalidRoomConfig):
		return http.StatusBadRequest, "bad_request", "bad request"
	default:
		return http.StatusInternalServerError, "internal_error", "internal error"
	}
}

func mapActionError(err error) (string, string) {
	switch {
	case errors.Is(err, room.ErrRoomNotFound), errors.Is(err, room.ErrRoomClosed):
		return "room_not_found", "room not found"
	case errors.Is(err, room.ErrUserNotInRoom):
		return "not_in_room", "not in room"
	case errors.Is(err, room.ErrGameNotStarted):
		return "game_not_started", "game not started"
	case errors.Is(err, room.ErrGameAlreadyStarted):
		return "game_already_started", "game already started"
	case errors.Is(err, room.ErrInvalidRoomConfig):
		return "bad_request", "bad request"
	case errors.Is(err, game.ErrInvalidGamePhase):
		return "invalid_game_phase", "invalid game phase"
	case errors.Is(err, game.ErrNotPlayerTurn):
		return "not_player_turn", "not player turn"
	case errors.Is(err, game.ErrInvalidBid):
		return "invalid_bid", "invalid bid"
	case errors.Is(err, game.ErrCannotPass):
		return "cannot_pass", "cannot pass"
	case errors.Is(err, game.ErrInvalidCardSet), errors.Is(err, game.ErrInvalidCardCode):
		return "invalid_card_set", "invalid card set"
	default:
		return "internal_error", "internal error"
	}
}

func decodeJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func decodePayload(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	return decodeJSON(raw, target)
}

func parseCardCodes(codes []string) ([]game.Card, error) {
	if len(codes) == 0 {
		return nil, nil
	}

	cards := make([]game.Card, 0, len(codes))
	for _, code := range codes {
		card, err := game.ParseCard(strings.TrimSpace(code))
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:      code,
		Message:   message,
		Data:      nil,
		RequestID: r.Header.Get("X-Request-ID"),
	})
}
