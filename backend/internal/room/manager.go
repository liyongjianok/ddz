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
	rooms    map[string]*Room
	userRoom map[string]string
	roomSeq  atomic.Uint64
}

// NewManager 创建一个内存版房间管理器。
func NewManager() *Manager {
	return &Manager{
		rooms:    make(map[string]*Room),
		userRoom: make(map[string]string),
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
		room := m.rooms[roomID]
		if room != nil && room.Status != RoomStatusClosed {
			return nil, -1, ErrUserAlreadyInActiveRoom
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
		JoinedAt:  now,
	})

	m.rooms[room.ID] = room
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
	defer m.mu.Unlock()

	if roomID, exists := m.userRoom[input.UserID]; exists {
		if roomID == input.RoomID {
			room := m.rooms[input.RoomID]
			if room == nil {
				delete(m.userRoom, input.UserID)
				return nil, -1, ErrRoomNotFound
			}
			for _, seat := range room.Seats {
				if seat.UserID == input.UserID {
					snapshot := room.Snapshot()
					return &snapshot, seat.SeatIndex, nil
				}
			}
		}
		return nil, -1, ErrUserAlreadyInActiveRoom
	}

	room, exists := m.rooms[input.RoomID]
	if !exists {
		return nil, -1, ErrRoomNotFound
	}
	if room.Status != RoomStatusWaiting {
		return nil, -1, ErrGameAlreadyStarted
	}
	if len(room.Seats) >= room.MaxPlayers {
		return nil, -1, ErrRoomFull
	}

	seatIndex, err := firstAvailableSeat(room, input.PreferredSeat)
	if err != nil {
		return nil, -1, err
	}

	now := time.Now().UTC()
	room.Seats = append(room.Seats, Seat{
		UserID:    input.UserID,
		SeatIndex: seatIndex,
		JoinedAt:  now,
	})
	room.UpdatedAt = now
	m.userRoom[input.UserID] = room.ID

	snapshot := room.Snapshot()
	return &snapshot, seatIndex, nil
}

// LeaveRoom 允许用户离开等待房间；空房间会被关闭并从管理器移除。
func (m *Manager) LeaveRoom(roomID string, userID string) (*Room, error) {
	if roomID == "" || userID == "" {
		return nil, ErrInvalidRoomConfig
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	room, exists := m.rooms[roomID]
	if !exists {
		return nil, ErrRoomNotFound
	}
	if room.Status != RoomStatusWaiting {
		return nil, ErrGameAlreadyStarted
	}

	seatIndex := -1
	for i, seat := range room.Seats {
		if seat.UserID == userID {
			seatIndex = i
			break
		}
	}
	if seatIndex < 0 {
		return nil, ErrUserNotInRoom
	}

	room.Seats = append(room.Seats[:seatIndex], room.Seats[seatIndex+1:]...)
	room.UpdatedAt = time.Now().UTC()
	delete(m.userRoom, userID)

	if len(room.Seats) == 0 {
		room.Status = RoomStatusClosed
		delete(m.rooms, roomID)
		snapshot := room.Snapshot()
		return &snapshot, nil
	}

	snapshot := room.Snapshot()
	return &snapshot, nil
}

// QuickStart 优先返回已有等待房间，否则创建新房间。
func (m *Manager) QuickStart(input QuickStartInput) (*Room, int, error) {
	if input.UserID == "" {
		return nil, -1, ErrInvalidRoomConfig
	}

	m.mu.Lock()
	if roomID, exists := m.userRoom[input.UserID]; exists {
		room := m.rooms[roomID]
		if room != nil && room.Status != RoomStatusClosed {
			for _, seat := range room.Seats {
				if seat.UserID == input.UserID {
					snapshot := room.Snapshot()
					m.mu.Unlock()
					return &snapshot, seat.SeatIndex, nil
				}
			}
		}
		delete(m.userRoom, input.UserID)
	}

	mode := normalizeMode(input.Mode)
	baseScore := normalizeBaseScore(input.BaseScore)
	for _, room := range m.rooms {
		if room.Status != RoomStatusWaiting {
			continue
		}
		if room.Mode != mode || room.BaseScore != baseScore {
			continue
		}
		if len(room.Seats) >= room.MaxPlayers {
			continue
		}

		seatIndex, err := firstAvailableSeat(room, nil)
		if err != nil {
			continue
		}

		now := time.Now().UTC()
		room.Seats = append(room.Seats, Seat{
			UserID:    input.UserID,
			SeatIndex: seatIndex,
			JoinedAt:  now,
		})
		room.UpdatedAt = now
		m.userRoom[input.UserID] = room.ID
		snapshot := room.Snapshot()
		m.mu.Unlock()
		return &snapshot, seatIndex, nil
	}
	m.mu.Unlock()

	return m.CreateRoom(CreateRoomInput{
		UserID:    input.UserID,
		BaseScore: baseScore,
		Mode:      mode,
	})
}

// GetRoom 返回房间当前快照。
func (m *Manager) GetRoom(roomID string) (*Room, error) {
	if roomID == "" {
		return nil, ErrRoomNotFound
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	room, exists := m.rooms[roomID]
	if !exists {
		return nil, ErrRoomNotFound
	}

	snapshot := room.Snapshot()
	return &snapshot, nil
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
