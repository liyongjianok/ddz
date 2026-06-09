package loadtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const seatsPerRoom = 3

// Config 定义负载冒烟测试参数。
type Config struct {
	BaseURL          string
	Mode             string
	BaseScore        int
	TotalConnections int
	Concurrency      int
	ReadyRooms       int
	HoldDuration     time.Duration
	HTTPTimeout      time.Duration
	ConnectTimeout   time.Duration
}

// Result 汇总一次负载冒烟执行结果。
type Result struct {
	StartedAt            time.Time     `json:"started_at"`
	CompletedAt          time.Time     `json:"completed_at"`
	Duration             time.Duration `json:"duration"`
	TargetConnections    int           `json:"target_connections"`
	TargetRooms          int           `json:"target_rooms"`
	ConnectedConnections int           `json:"connected_connections"`
	CreatedRooms         int           `json:"created_rooms"`
	JoinedPlayers        int           `json:"joined_players"`
	ReadyActions         int           `json:"ready_actions"`
	FailureCount         int           `json:"failure_count"`
	Failures             []string      `json:"failures,omitempty"`
}

type roomPlan struct {
	Index       int
	PlayerCount int
}

type smokeClient struct {
	UserID string
	RoomID string
	Conn   *websocket.Conn

	readDone chan struct{}
}

type roomOutcome struct {
	clients      []*smokeClient
	createdRooms int
	joinedUsers  int
	readyActions int
	failures     []string
}

type apiEnvelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type guestLoginData struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	AccessToken string `json:"access_token"`
}

type roomAccessData struct {
	RoomID string `json:"room_id"`
}

type wsEnvelope struct {
	Type string `json:"type"`
}

type readyAckPayload struct {
	Ok bool `json:"ok"`
}

type readyMessage struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id"`
	Seq       uint64      `json:"seq"`
	Payload   readyAction `json:"payload"`
}

type readyAction struct {
	Ready bool `json:"ready"`
}

type ackEnvelope struct {
	Type      string          `json:"type"`
	RequestID *string         `json:"request_id"`
	Payload   json.RawMessage `json:"payload"`
}

// Run 执行一次负载冒烟测试。
func Run(ctx context.Context, cfg Config) (Result, error) {
	cfg = normalizeConfig(cfg)
	if err := validateConfig(cfg); err != nil {
		return Result{}, err
	}

	result := Result{
		StartedAt:         time.Now().UTC(),
		TargetConnections: cfg.TotalConnections,
	}

	plans := buildRoomPlans(cfg.TotalConnections)
	result.TargetRooms = len(plans)

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	dialer := &websocket.Dialer{HandshakeTimeout: cfg.ConnectTimeout}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, cfg.Concurrency)
		clients []*smokeClient
	)

	for _, plan := range plans {
		plan := plan
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			outcome := runRoom(ctx, httpClient, dialer, cfg, plan)

			mu.Lock()
			defer mu.Unlock()
			result.CreatedRooms += outcome.createdRooms
			result.JoinedPlayers += outcome.joinedUsers
			result.ReadyActions += outcome.readyActions
			result.FailureCount += len(outcome.failures)
			result.Failures = append(result.Failures, outcome.failures...)
			for _, client := range outcome.clients {
				clients = append(clients, client)
			}
		}()
	}

	wg.Wait()

	for _, client := range clients {
		client.startReader()
	}

	result.ConnectedConnections = len(clients)

	timer := time.NewTimer(cfg.HoldDuration)
	select {
	case <-ctx.Done():
		timer.Stop()
	case <-timer.C:
	}

	for _, client := range clients {
		client.close()
	}

	result.CompletedAt = time.Now().UTC()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	if result.FailureCount > 0 || result.ConnectedConnections != result.TargetConnections {
		return result, fmt.Errorf("load smoke finished with %d failure(s), connected %d/%d", result.FailureCount, result.ConnectedConnections, result.TargetConnections)
	}
	return result, nil
}

func normalizeConfig(cfg Config) Config {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://127.0.0.1:8080"
	}
	if cfg.Mode == "" {
		cfg.Mode = "classic"
	}
	if cfg.BaseScore <= 0 {
		cfg.BaseScore = 1
	}
	if cfg.TotalConnections <= 0 {
		cfg.TotalConnections = 30
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 20
	}
	if cfg.HoldDuration <= 0 {
		cfg.HoldDuration = 10 * time.Second
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.ReadyRooms < 0 {
		cfg.ReadyRooms = 0
	}
	return cfg
}

func validateConfig(cfg Config) error {
	if _, err := url.Parse(cfg.BaseURL); err != nil {
		return fmt.Errorf("invalid base url: %w", err)
	}
	if cfg.TotalConnections <= 0 {
		return errors.New("total connections must be positive")
	}
	if cfg.Concurrency <= 0 {
		return errors.New("concurrency must be positive")
	}
	return nil
}

func buildRoomPlans(totalConnections int) []roomPlan {
	plans := make([]roomPlan, 0, (totalConnections+seatsPerRoom-1)/seatsPerRoom)
	remaining := totalConnections
	for idx := 0; remaining > 0; idx++ {
		size := seatsPerRoom
		if remaining < seatsPerRoom {
			size = remaining
		}
		plans = append(plans, roomPlan{
			Index:       idx,
			PlayerCount: size,
		})
		remaining -= size
	}
	return plans
}

func runRoom(ctx context.Context, httpClient *http.Client, dialer *websocket.Dialer, cfg Config, plan roomPlan) roomOutcome {
	var outcome roomOutcome

	hostName := fmt.Sprintf("smoke_%04d_host", plan.Index+1)
	hostUserID, hostToken, err := guestLogin(ctx, httpClient, cfg.BaseURL, hostName)
	if err != nil {
		outcome.failures = append(outcome.failures, fmt.Sprintf("room[%d] host login: %v", plan.Index, err))
		return outcome
	}

	roomID, err := createRoom(ctx, httpClient, cfg, hostToken)
	if err != nil {
		outcome.failures = append(outcome.failures, fmt.Sprintf("room[%d] create room: %v", plan.Index, err))
		return outcome
	}
	outcome.createdRooms = 1

	hostClient, err := connectRoom(ctx, dialer, cfg.BaseURL, roomID, hostToken, hostUserID)
	if err != nil {
		outcome.failures = append(outcome.failures, fmt.Sprintf("room[%d] host ws: %v", plan.Index, err))
		return outcome
	}
	outcome.clients = append(outcome.clients, hostClient)

	for seat := 1; seat < plan.PlayerCount; seat++ {
		playerName := fmt.Sprintf("smoke_%04d_guest_%d", plan.Index+1, seat)
		userID, token, loginErr := guestLogin(ctx, httpClient, cfg.BaseURL, playerName)
		if loginErr != nil {
			outcome.failures = append(outcome.failures, fmt.Sprintf("room[%d] guest[%d] login: %v", plan.Index, seat, loginErr))
			continue
		}

		if joinErr := joinRoom(ctx, httpClient, cfg.BaseURL, roomID, token, seat); joinErr != nil {
			outcome.failures = append(outcome.failures, fmt.Sprintf("room[%d] guest[%d] join: %v", plan.Index, seat, joinErr))
			continue
		}
		outcome.joinedUsers++

		client, connErr := connectRoom(ctx, dialer, cfg.BaseURL, roomID, token, userID)
		if connErr != nil {
			outcome.failures = append(outcome.failures, fmt.Sprintf("room[%d] guest[%d] ws: %v", plan.Index, seat, connErr))
			continue
		}
		outcome.clients = append(outcome.clients, client)
	}

	if plan.Index < cfg.ReadyRooms && len(outcome.clients) > 0 {
		if err := sendReady(outcome.clients[0].Conn); err != nil {
			outcome.failures = append(outcome.failures, fmt.Sprintf("room[%d] ready action: %v", plan.Index, err))
		} else {
			outcome.readyActions++
		}
	}

	return outcome
}

func guestLogin(ctx context.Context, httpClient *http.Client, baseURL string, displayName string) (string, string, error) {
	payload := map[string]any{"display_name": displayName}
	var data guestLoginData
	if err := postJSON(ctx, httpClient, baseURL+"/api/v1/auth/guest", "", payload, &data); err != nil {
		return "", "", err
	}
	return data.User.ID, data.AccessToken, nil
}

func createRoom(ctx context.Context, httpClient *http.Client, cfg Config, token string) (string, error) {
	payload := map[string]any{
		"mode":       cfg.Mode,
		"base_score": cfg.BaseScore,
		"private":    false,
	}
	var data roomAccessData
	if err := postJSON(ctx, httpClient, cfg.BaseURL+"/api/v1/rooms", token, payload, &data); err != nil {
		return "", err
	}
	return data.RoomID, nil
}

func joinRoom(ctx context.Context, httpClient *http.Client, baseURL string, roomID string, token string, preferredSeat int) error {
	payload := map[string]any{
		"preferred_seat": preferredSeat,
	}
	return postJSON(ctx, httpClient, baseURL+"/api/v1/rooms/"+roomID+"/join", token, payload, &struct{}{})
}

func connectRoom(ctx context.Context, dialer *websocket.Dialer, baseURL string, roomID string, token string, userID string) (*smokeClient, error) {
	wsURL, err := buildWSURL(baseURL, roomID, token)
	if err != nil {
		return nil, err
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var envelope wsEnvelope
	if err := conn.ReadJSON(&envelope); err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetReadDeadline(time.Time{})
	if envelope.Type != "room.snapshot" {
		_ = conn.Close()
		return nil, fmt.Errorf("unexpected first ws event: %s", envelope.Type)
	}

	return &smokeClient{
		UserID:   userID,
		RoomID:   roomID,
		Conn:     conn,
		readDone: make(chan struct{}),
	}, nil
}

func buildWSURL(baseURL string, roomID string, token string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	parsed.Path = "/ws/v1/rooms/" + roomID
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func postJSON(ctx context.Context, httpClient *http.Client, endpoint string, token string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK || envelope.Code != "ok" {
		return fmt.Errorf("status=%d code=%s message=%s", resp.StatusCode, envelope.Code, envelope.Message)
	}
	if target == nil {
		return nil
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	return json.Unmarshal(envelope.Data, target)
}

func sendReady(conn *websocket.Conn) error {
	if conn == nil {
		return errors.New("nil websocket connection")
	}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteJSON(readyMessage{
		Type:      "room.ready",
		RequestID: "smoke-ready-1",
		Seq:       1,
		Payload: readyAction{
			Ready: true,
		},
	}); err != nil {
		return err
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	for {
		var envelope ackEnvelope
		if err := conn.ReadJSON(&envelope); err != nil {
			return err
		}
		switch envelope.Type {
		case "ack":
			var payload readyAckPayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				return err
			}
			if !payload.Ok {
				return errors.New("ready ack not ok")
			}
			return nil
		case "error":
			return fmt.Errorf("ready rejected")
		}
	}
}

func (c *smokeClient) startReader() {
	if c == nil || c.Conn == nil || c.readDone == nil {
		return
	}

	go func() {
		defer close(c.readDone)
		for {
			if _, _, err := c.Conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (c *smokeClient) close() {
	if c == nil || c.Conn == nil {
		return
	}

	_ = c.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "smoke done"), time.Now().Add(time.Second))
	_ = c.Conn.Close()
	if c.readDone != nil {
		<-c.readDone
	}
}
