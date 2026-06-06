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

type roomPlayerReadyEvent struct {
	UserID    string `json:"user_id"`
	SeatIndex int    `json:"seat_index"`
	Ready     bool   `json:"ready"`
}

type gameBidPlacedEvent struct {
	UserID        string    `json:"user_id"`
	SeatIndex     int       `json:"seat_index"`
	Score         int       `json:"score"`
	NextSeatIndex int       `json:"next_seat_index"`
	DeadlineAt    time.Time `json:"deadline_at,omitempty"`
}

type gameLandlordDecidedEvent struct {
	LandlordSeatIndex int       `json:"landlord_seat_index"`
	LandlordUserID    string    `json:"landlord_user_id"`
	BottomCards       []string  `json:"bottom_cards"`
	Multiplier        int       `json:"multiplier"`
	CurrentSeatIndex  int       `json:"current_seat_index"`
	DeadlineAt        time.Time `json:"deadline_at,omitempty"`
}

type gameCardsPlayedEvent struct {
	UserID         string               `json:"user_id"`
	SeatIndex      int                  `json:"seat_index"`
	Cards          []string             `json:"cards"`
	CardGroup      gameCardsPlayedGroup `json:"card_group"`
	RemainingCount int                  `json:"remaining_count"`
	NextSeatIndex  int                  `json:"next_seat_index"`
	DeadlineAt     time.Time            `json:"deadline_at,omitempty"`
}

type gameCardsPlayedGroup struct {
	Type        string   `json:"type"`
	Rank        string   `json:"rank"`
	Length      int      `json:"length"`
	Attachments []string `json:"attachments,omitempty"`
}

type gameMyHandUpdatedEvent struct {
	Cards []string `json:"cards"`
}

type gamePlayerPassedEvent struct {
	UserID        string    `json:"user_id"`
	SeatIndex     int       `json:"seat_index"`
	NextSeatIndex int       `json:"next_seat_index"`
	DeadlineAt    time.Time `json:"deadline_at,omitempty"`
}

type gameEndedEvent struct {
	WinnerSide      string                     `json:"winner_side"`
	WinnerUserID    string                     `json:"winner_user_id"`
	Settlements     []gameEndedSettlementEvent `json:"settlements"`
	FinalMultiplier int                        `json:"final_multiplier"`
}

type gameEndedSettlementEvent struct {
	UserID     string `json:"user_id"`
	SeatIndex  int    `json:"seat_index"`
	Role       string `json:"role"`
	ScoreDelta int    `json:"score_delta"`
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

func (g *Gateway) roomClients(roomID string) []*clientConn {
	g.mu.Lock()
	defer g.mu.Unlock()

	roomClients := g.rooms[roomID]
	if len(roomClients) == 0 {
		return nil
	}

	clients := make([]*clientConn, 0, len(roomClients))
	for _, client := range roomClients {
		clients = append(clients, client)
	}
	return clients
}

func (g *Gateway) sendToUser(roomID string, userID string, messageType string, requestID *string, payload any) {
	clients := g.roomClients(roomID)
	for _, client := range clients {
		if client.userID != userID {
			continue
		}
		if err := client.sendMessage(messageType, requestID, payload); err != nil {
			g.unregister(client)
		}
	}
}

func (g *Gateway) broadcast(roomID string, messageType string, requestID *string, payload any) {
	clients := g.roomClients(roomID)
	for _, client := range clients {
		if err := client.sendMessage(messageType, requestID, payload); err != nil {
			g.unregister(client)
		}
	}
}

func (g *Gateway) broadcastRoomSnapshots(roomID string) {
	clients := g.roomClients(roomID)
	for _, client := range clients {
		snapshot, err := g.roomManager.GetRoomSnapshot(roomID, client.userID)
		if err != nil {
			continue
		}
		if err := client.sendMessage("room.snapshot", nil, snapshot); err != nil {
			g.unregister(client)
		}
	}
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

	currentRoom, seatIndex, started, err := c.gateway.roomManager.Ready(room.ReadyInput{
		RoomID: c.roomID,
		UserID: c.userID,
		Ready:  payload.Ready,
	})
	if err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	if err := c.sendAck(message.RequestID); err != nil {
		return err
	}

	c.gateway.broadcast(c.roomID, "room.player_ready", message.RequestID, roomPlayerReadyEvent{
		UserID:    c.userID,
		SeatIndex: seatIndex,
		Ready:     payload.Ready,
	})
	if started && currentRoom != nil {
		c.gateway.broadcastRoomSnapshots(c.roomID)
	}

	return nil
}

func (c *clientConn) handleBid(message clientMessage) error {
	var payload bidPayload
	if err := decodePayload(message.Payload, &payload); err != nil {
		return c.sendError(message.RequestID, "bad_request", "bad request")
	}

	currentRoom, err := c.gateway.roomManager.Bid(room.BidInput{
		RoomID: c.roomID,
		UserID: c.userID,
		Score:  payload.Score,
	})
	if err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	if err := c.sendAck(message.RequestID); err != nil {
		return err
	}
	if currentRoom == nil || currentRoom.CurrentGame == nil {
		return nil
	}

	seatIndex, ok := findSeatIndexByUserID(currentRoom.CurrentGame.Players, c.userID)
	if !ok {
		return nil
	}
	c.gateway.broadcast(c.roomID, "game.bid_placed", message.RequestID, gameBidPlacedEvent{
		UserID:        c.userID,
		SeatIndex:     seatIndex,
		Score:         payload.Score,
		NextSeatIndex: currentRoom.CurrentGame.CurrentSeatIndex,
		DeadlineAt:    currentRoom.DeadlineAt,
	})

	if currentRoom.CurrentGame.Phase == game.GamePhasePlaying && currentRoom.CurrentGame.LandlordSeatIndex >= 0 {
		landlordPlayer := currentRoom.CurrentGame.Players[currentRoom.CurrentGame.LandlordSeatIndex]
		c.gateway.broadcast(c.roomID, "game.landlord_decided", message.RequestID, gameLandlordDecidedEvent{
			LandlordSeatIndex: currentRoom.CurrentGame.LandlordSeatIndex,
			LandlordUserID:    landlordPlayer.UserID,
			BottomCards:       cardCodes(currentRoom.CurrentGame.BottomCards),
			Multiplier:        currentRoom.CurrentGame.Multiplier,
			CurrentSeatIndex:  currentRoom.CurrentGame.CurrentSeatIndex,
			DeadlineAt:        currentRoom.DeadlineAt,
		})

		c.gateway.sendHandUpdated(c.roomID, landlordPlayer.UserID, message.RequestID)
	}

	return nil
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

	currentRoom, err := c.gateway.roomManager.PlayCards(room.PlayCardsInput{
		RoomID: c.roomID,
		UserID: c.userID,
		Cards:  cards,
	})
	if err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	if err := c.sendAck(message.RequestID); err != nil {
		return err
	}
	if currentRoom == nil || currentRoom.CurrentGame == nil || currentRoom.CurrentGame.LastPlay == nil {
		return nil
	}

	lastPlay := currentRoom.CurrentGame.LastPlay
	remainingCount, _ := findRemainingCountByUserID(currentRoom.CurrentGame.Players, c.userID)
	c.gateway.broadcast(c.roomID, "game.cards_played", message.RequestID, gameCardsPlayedEvent{
		UserID:    c.userID,
		SeatIndex: lastPlay.SeatIndex,
		Cards:     cardCodes(lastPlay.Cards),
		CardGroup: gameCardsPlayedGroup{
			Type:        string(lastPlay.Group.Type),
			Rank:        lastPlay.Group.PrimaryRank.String(),
			Length:      lastPlay.Group.Length,
			Attachments: cardCodes(lastPlay.Group.Attachments),
		},
		RemainingCount: remainingCount,
		NextSeatIndex:  currentRoom.CurrentGame.CurrentSeatIndex,
		DeadlineAt:     currentRoom.DeadlineAt,
	})
	c.gateway.sendHandUpdated(c.roomID, c.userID, message.RequestID)

	if currentRoom.CurrentGame.Phase == game.GamePhaseEnded || currentRoom.Status == room.RoomStatusSettling {
		c.gateway.broadcastGameEnded(currentRoom, message.RequestID)
	}

	return nil
}

func (c *clientConn) handlePass(message clientMessage) error {
	var payload struct{}
	if err := decodePayload(message.Payload, &payload); err != nil {
		return c.sendError(message.RequestID, "bad_request", "bad request")
	}

	currentRoom, err := c.gateway.roomManager.Pass(room.PassInput{
		RoomID: c.roomID,
		UserID: c.userID,
	})
	if err != nil {
		return c.sendActionError(message.RequestID, err)
	}

	if err := c.sendAck(message.RequestID); err != nil {
		return err
	}
	if currentRoom == nil || currentRoom.CurrentGame == nil {
		return nil
	}

	seatIndex, ok := findSeatIndexByUserID(currentRoom.CurrentGame.Players, c.userID)
	if !ok {
		return nil
	}
	c.gateway.broadcast(c.roomID, "game.player_passed", message.RequestID, gamePlayerPassedEvent{
		UserID:        c.userID,
		SeatIndex:     seatIndex,
		NextSeatIndex: currentRoom.CurrentGame.CurrentSeatIndex,
		DeadlineAt:    currentRoom.DeadlineAt,
	})

	return nil
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

func (g *Gateway) sendHandUpdated(roomID string, userID string, requestID *string) {
	snapshot, err := g.roomManager.GetRoomSnapshot(roomID, userID)
	if err != nil {
		return
	}

	g.sendToUser(roomID, userID, "game.my_hand_updated", requestID, gameMyHandUpdatedEvent{
		Cards: snapshot.Me.Hand,
	})
}

func (g *Gateway) broadcastGameEnded(currentRoom *room.Room, requestID *string) {
	if currentRoom == nil || currentRoom.CurrentGame == nil || currentRoom.CurrentGame.Settlement == nil {
		return
	}

	winnerUserID := ""
	for _, player := range currentRoom.CurrentGame.Settlement.Players {
		if player.IsWinner {
			winnerUserID = player.UserID
			break
		}
	}

	settlements := make([]gameEndedSettlementEvent, 0, len(currentRoom.CurrentGame.Settlement.Players))
	for _, player := range currentRoom.CurrentGame.Settlement.Players {
		settlements = append(settlements, gameEndedSettlementEvent{
			UserID:     player.UserID,
			SeatIndex:  player.SeatIndex,
			Role:       string(player.Role),
			ScoreDelta: player.DeltaScore,
		})
	}

	g.broadcast(currentRoom.ID, "game.ended", requestID, gameEndedEvent{
		WinnerSide:      string(currentRoom.CurrentGame.Settlement.WinnerSide),
		WinnerUserID:    winnerUserID,
		Settlements:     settlements,
		FinalMultiplier: currentRoom.CurrentGame.Settlement.Multiplier,
	})
}

func cardCodes(cards []game.Card) []string {
	if len(cards) == 0 {
		return nil
	}

	result := make([]string, 0, len(cards))
	for _, card := range cards {
		result = append(result, card.Code())
	}
	return result
}

func findSeatIndexByUserID(players []game.PlayerState, userID string) (int, bool) {
	for _, player := range players {
		if player.UserID == userID {
			return player.SeatIndex, true
		}
	}
	return -1, false
}

func findRemainingCountByUserID(players []game.PlayerState, userID string) (int, bool) {
	for _, player := range players {
		if player.UserID == userID {
			return player.RemainingCount, true
		}
	}
	return 0, false
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
