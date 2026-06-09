package redisstore

import (
	"context"
	"sync"
	"time"
)

type memoryEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// MemoryStore 是 Redis Store 的内存实现，便于本地开发和测试。
type MemoryStore struct {
	mu                sync.RWMutex
	playerConnections map[string]memoryEntry[PlayerConnection]
	reconnectStates   map[string]memoryEntry[ReconnectState]
	roomSnapshots     map[string]memoryEntry[RoomSnapshotState]
	now               func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		playerConnections: make(map[string]memoryEntry[PlayerConnection]),
		reconnectStates:   make(map[string]memoryEntry[ReconnectState]),
		roomSnapshots:     make(map[string]memoryEntry[RoomSnapshotState]),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *MemoryStore) SavePlayerConnection(_ context.Context, connection PlayerConnection, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.playerConnections[playerConnectionKey(connection.UserID, connection.RoomID)] = memoryEntry[PlayerConnection]{
		value:     connection,
		expiresAt: s.expireAt(ttl),
	}
	return nil
}

func (s *MemoryStore) GetPlayerConnection(_ context.Context, userID string, roomID string) (PlayerConnection, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.playerConnections[playerConnectionKey(userID, roomID)]
	if !ok || s.isExpired(entry.expiresAt) {
		delete(s.playerConnections, playerConnectionKey(userID, roomID))
		return PlayerConnection{}, false, nil
	}
	return entry.value, true, nil
}

func (s *MemoryStore) DeletePlayerConnection(_ context.Context, userID string, roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.playerConnections, playerConnectionKey(userID, roomID))
	return nil
}

func (s *MemoryStore) SaveReconnectState(_ context.Context, state ReconnectState, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reconnectStates[reconnectStateKey(state.UserID, state.RoomID)] = memoryEntry[ReconnectState]{
		value:     state,
		expiresAt: s.expireAt(ttl),
	}
	return nil
}

func (s *MemoryStore) GetReconnectState(_ context.Context, userID string, roomID string) (ReconnectState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.reconnectStates[reconnectStateKey(userID, roomID)]
	if !ok || s.isExpired(entry.expiresAt) {
		delete(s.reconnectStates, reconnectStateKey(userID, roomID))
		return ReconnectState{}, false, nil
	}
	return entry.value, true, nil
}

func (s *MemoryStore) DeleteReconnectState(_ context.Context, userID string, roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.reconnectStates, reconnectStateKey(userID, roomID))
	return nil
}

func (s *MemoryStore) SaveRoomSnapshot(_ context.Context, snapshot RoomSnapshotState, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.roomSnapshots[roomSnapshotKey(snapshot.RoomID)] = memoryEntry[RoomSnapshotState]{
		value:     snapshot,
		expiresAt: s.expireAt(ttl),
	}
	return nil
}

func (s *MemoryStore) GetRoomSnapshot(_ context.Context, roomID string) (RoomSnapshotState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.roomSnapshots[roomSnapshotKey(roomID)]
	if !ok || s.isExpired(entry.expiresAt) {
		delete(s.roomSnapshots, roomSnapshotKey(roomID))
		return RoomSnapshotState{}, false, nil
	}
	return entry.value, true, nil
}

func (s *MemoryStore) DeleteRoomSnapshot(_ context.Context, roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.roomSnapshots, roomSnapshotKey(roomID))
	return nil
}

func (s *MemoryStore) expireAt(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return s.now().Add(ttl)
}

func (s *MemoryStore) isExpired(expiresAt time.Time) bool {
	return !expiresAt.IsZero() && !expiresAt.After(s.now())
}

func playerConnectionKey(userID string, roomID string) string {
	return userID + ":" + roomID
}

func reconnectStateKey(userID string, roomID string) string {
	return userID + ":" + roomID
}

func roomSnapshotKey(roomID string) string {
	return roomID
}
