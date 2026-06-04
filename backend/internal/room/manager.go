package room

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"ddz/backend/internal/game"
)

var ErrInvalidRoomConfig = errors.New("invalid room config")
var ErrRoomNotFound = errors.New("room not found")
var ErrSeatUnavailable = errors.New("seat unavailable")
var ErrRoomFull = errors.New("room full")
var ErrUserAlreadyInActiveRoom = errors.New("user already in active room")
var ErrGameAlreadyStarted = errors.New("game already started")
var ErrUserNotInRoom = errors.New("user not in room")

type RoomStatus string

const (
	RoomStatusWaiting  RoomStatus = "waiting"
	RoomStatusPlaying  RoomStatus = "playing"
	RoomStatusSettling RoomStatus = "settling"
	RoomStatusClosed   RoomStatus = "closed"
)

const DefaultMode = "classic"

// CreateRoomInput 描述创建房间时所需的参数。
type CreateRoomInput struct {
	UserID        string
	PreferredSeat *int
	BaseScore     int
	Mode          string
}

// JoinRoomInput 描述加入房间时所需的参数。
type JoinRoomInput struct {
	RoomID        string
	UserID        string
	PreferredSeat *int
}

// ReadyInput 描述玩家修改准备状态时所需的参数。
type ReadyInput struct {
	RoomID string
	UserID string
	Ready  bool
}

// QuickStartInput 描述快速开始匹配所需的参数。
type QuickStartInput struct {
	UserID    string
	BaseScore int
	Mode      string
}

// Seat 表示等待房间中的一个座位。
type Seat struct {
	UserID    string
	SeatIndex int
	IsRobot   bool
	Ready     bool
	JoinedAt  time.Time
}

// Room 表示等待或进行中的房间聚合。
type Room struct {
	ID          string
	Mode        string
	Status      RoomStatus
	BaseScore   int
	MaxPlayers  int
	Seats       []Seat
	CurrentGame *game.Game
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Snapshot 返回房间的只读副本，避免调用方直接修改内部状态。
func (r *Room) Snapshot() Room {
	seats := make([]Seat, len(r.Seats))
	copy(seats, r.Seats)

	return Room{
		ID:          r.ID,
		Mode:        r.Mode,
		Status:      r.Status,
		BaseScore:   r.BaseScore,
		MaxPlayers:  r.MaxPlayers,
		Seats:       seats,
		CurrentGame: r.CurrentGame,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// Manager 负责管理活跃房间及用户与房间的归属关系。
type Manager struct {
	mu       sync.RWMutex
	rooms    map[string]*RoomActor
	userRoom map[string]string
	roomSeq  atomic.Uint64
	rng      game.RNG
}

// NewManager 创建一个内存版房间管理器。
func NewManager() *Manager {
	return NewManagerWithRNG(nil)
}

// NewManagerWithRNG 创建一个可注入随机源的房间管理器。
func NewManagerWithRNG(rng game.RNG) *Manager {
	return &Manager{
		rooms:    make(map[string]*RoomActor),
		userRoom: make(map[string]string),
		rng:      rng,
	}
}

// CreateRoom 创建等待房间，并让创建者占据一个座位。
func (m *Manager) CreateRoom(input CreateRoomInput) (*Room, int, error) {
	if err := validateCreateRoomInput(input); err != nil {
		return nil, -1, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if roomID, exists := m.userRoom[input.UserID]; exists {
		actor := m.rooms[roomID]
		if actor != nil {
			room, err := actor.Snapshot()
			if err != nil {
				return nil, -1, err
			}
			if room.Status != RoomStatusClosed {
				return nil, -1, ErrUserAlreadyInActiveRoom
			}
		}
		delete(m.userRoom, input.UserID)
	}

	roomID := m.nextRoomID()
	now := time.Now().UTC()
	room := &Room{
		ID:         roomID,
		Mode:       normalizeMode(input.Mode),
		Status:     RoomStatusWaiting,
		BaseScore:  normalizeBaseScore(input.BaseScore),
		MaxPlayers: game.PlayerCount,
		Seats:      make([]Seat, 0, game.PlayerCount),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	seatIndex, err := firstAvailableSeat(room, input.PreferredSeat)
	if err != nil {
		return nil, -1, err
	}

	room.Seats = append(room.Seats, Seat{
		UserID:    input.UserID,
		SeatIndex: seatIndex,
		Ready:     false,
		JoinedAt:  now,
	})

	m.rooms[room.ID] = NewRoomActor(room, m.rng)
	m.userRoom[input.UserID] = room.ID

	snapshot := room.Snapshot()
	return &snapshot, seatIndex, nil
}

// JoinRoom 将用户加入一个等待中的房间。
func (m *Manager) JoinRoom(input JoinRoomInput) (*Room, int, error) {
	if input.RoomID == "" || input.UserID == "" {
		return nil, -1, ErrInvalidRoomConfig
	}

	m.mu.Lock()
	if roomID, exists := m.userRoom[input.UserID]; exists {
		if roomID == input.RoomID {
			actor := m.rooms[input.RoomID]
			if actor == nil {
				delete(m.userRoom, input.UserID)
				m.mu.Unlock()
				return nil, -1, ErrRoomNotFound
			}
			room, seatIndex, _, err := actor.Join(input.UserID, input.PreferredSeat)
			m.mu.Unlock()
			if err != nil {
				return nil, -1, err
			}
			return &room, seatIndex, nil
		}
		m.mu.Unlock()
		return nil, -1, ErrUserAlreadyInActiveRoom
	}

	actor, exists := m.rooms[input.RoomID]
	if !exists {
		m.mu.Unlock()
		return nil, -1, ErrRoomNotFound
	}

	room, seatIndex, _, err := actor.Join(input.UserID, input.PreferredSeat)
	if err != nil {
		m.mu.Unlock()
		return nil, -1, err
	}

	m.userRoom[input.UserID] = input.RoomID
	m.mu.Unlock()
	return &room, seatIndex, nil
}

// LeaveRoom 允许用户离开等待房间；空房间会被关闭并从管理器移除。
func (m *Manager) LeaveRoom(roomID string, userID string) (*Room, error) {
	if roomID == "" || userID == "" {
		return nil, ErrInvalidRoomConfig
	}

	m.mu.Lock()
	actor, exists := m.rooms[roomID]
	if !exists {
		m.mu.Unlock()
		return nil, ErrRoomNotFound
	}

	room, _, _, err := actor.Leave(userID)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}

	delete(m.userRoom, userID)
	if room.Status == RoomStatusClosed {
		delete(m.rooms, roomID)
	}
	m.mu.Unlock()

	return &room, nil
}

// QuickStart 优先返回已有等待房间，否则创建新房间。
func (m *Manager) QuickStart(input QuickStartInput) (*Room, int, error) {
	if input.UserID == "" {
		return nil, -1, ErrInvalidRoomConfig
	}

	m.mu.Lock()
	if roomID, exists := m.userRoom[input.UserID]; exists {
		actor := m.rooms[roomID]
		if actor != nil {
			room, err := actor.Snapshot()
			if err == nil && room.Status != RoomStatusClosed {
				for _, seat := range room.Seats {
					if seat.UserID == input.UserID {
						m.mu.Unlock()
						return &room, seat.SeatIndex, nil
					}
				}
			}
		}
		delete(m.userRoom, input.UserID)
	}

	mode := normalizeMode(input.Mode)
	baseScore := normalizeBaseScore(input.BaseScore)
	for roomID, actor := range m.rooms {
		room, err := actor.Snapshot()
		if err != nil {
			continue
		}
		if room.Status != RoomStatusWaiting {
			continue
		}
		if room.Mode != mode || room.BaseScore != baseScore {
			continue
		}
		if len(room.Seats) >= room.MaxPlayers {
			continue
		}

		joinedRoom, seatIndex, _, err := actor.Join(input.UserID, nil)
		if err != nil {
			continue
		}
		m.userRoom[input.UserID] = roomID
		m.mu.Unlock()
		return &joinedRoom, seatIndex, nil
	}
	m.mu.Unlock()

	return m.CreateRoom(CreateRoomInput{
		UserID:    input.UserID,
		BaseScore: baseScore,
		Mode:      mode,
	})
}

// Ready 修改等待房间中的玩家准备状态；房满且全员准备后自动开局。
func (m *Manager) Ready(input ReadyInput) (*Room, int, bool, error) {
	if input.RoomID == "" || input.UserID == "" {
		return nil, -1, false, ErrInvalidRoomConfig
	}

	m.mu.RLock()
	actor, exists := m.rooms[input.RoomID]
	m.mu.RUnlock()
	if !exists {
		return nil, -1, false, ErrRoomNotFound
	}

	room, seatIndex, started, err := actor.Ready(input.UserID, input.Ready)
	if err != nil {
		return nil, -1, false, err
	}

	return &room, seatIndex, started, nil
}

// GetRoom 返回房间当前快照。
func (m *Manager) GetRoom(roomID string) (*Room, error) {
	if roomID == "" {
		return nil, ErrRoomNotFound
	}

	m.mu.RLock()
	actor, exists := m.rooms[roomID]
	m.mu.RUnlock()
	if !exists {
		return nil, ErrRoomNotFound
	}

	room, err := actor.Snapshot()
	if err != nil {
		return nil, err
	}

	return &room, nil
}

func (m *Manager) nextRoomID() string {
	seq := m.roomSeq.Add(1)
	return fmt.Sprintf("r_%06d", seq)
}

func validateCreateRoomInput(input CreateRoomInput) error {
	if input.UserID == "" {
		return ErrInvalidRoomConfig
	}
	if input.PreferredSeat != nil && (*input.PreferredSeat < 0 || *input.PreferredSeat >= game.PlayerCount) {
		return ErrSeatUnavailable
	}
	return nil
}

func normalizeMode(mode string) string {
	if mode == "" {
		return DefaultMode
	}
	return mode
}

func normalizeBaseScore(baseScore int) int {
	if baseScore <= 0 {
		return 1
	}
	return baseScore
}

func firstAvailableSeat(room *Room, preferredSeat *int) (int, error) {
	occupied := make(map[int]struct{}, len(room.Seats))
	for _, seat := range room.Seats {
		occupied[seat.SeatIndex] = struct{}{}
	}

	if preferredSeat != nil {
		seatIndex := *preferredSeat
		if seatIndex < 0 || seatIndex >= room.MaxPlayers {
			return -1, ErrSeatUnavailable
		}
		if _, exists := occupied[seatIndex]; exists {
			return -1, ErrSeatUnavailable
		}
		return seatIndex, nil
	}

	for seatIndex := 0; seatIndex < room.MaxPlayers; seatIndex++ {
		if _, exists := occupied[seatIndex]; !exists {
			return seatIndex, nil
		}
	}

	return -1, ErrRoomFull
}
