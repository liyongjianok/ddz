package redisstore

import (
	"context"
	"time"
)

// PlayerConnection 保存玩家在房间中的连接映射。
type PlayerConnection struct {
	UserID       string
	RoomID       string
	ConnectionID string
	UpdatedAt    time.Time
}

// ReconnectState 保存短期可恢复的重连元数据。
type ReconnectState struct {
	UserID         string
	RoomID         string
	SeatIndex      int
	GameID         string
	DisconnectedAt time.Time
	ExpiresAt      time.Time
}

// RoomSnapshotState 保存活跃房间的最新快照。
type RoomSnapshotState struct {
	RoomID    string
	Payload   []byte
	UpdatedAt time.Time
	ExpiresAt time.Time
}

// Store 定义 Redis 能力边界，后续可替换为真实 Redis 实现。
type Store interface {
	SavePlayerConnection(ctx context.Context, connection PlayerConnection, ttl time.Duration) error
	GetPlayerConnection(ctx context.Context, userID string, roomID string) (PlayerConnection, bool, error)
	DeletePlayerConnection(ctx context.Context, userID string, roomID string) error

	SaveReconnectState(ctx context.Context, state ReconnectState, ttl time.Duration) error
	GetReconnectState(ctx context.Context, userID string, roomID string) (ReconnectState, bool, error)
	DeleteReconnectState(ctx context.Context, userID string, roomID string) error

	SaveRoomSnapshot(ctx context.Context, snapshot RoomSnapshotState, ttl time.Duration) error
	GetRoomSnapshot(ctx context.Context, roomID string) (RoomSnapshotState, bool, error)
	DeleteRoomSnapshot(ctx context.Context, roomID string) error
}
