package profile

import "context"

// Store 定义玩家资料存储边界。
type Store interface {
	EnsureProfile(ctx context.Context, userID string) (PlayerProfile, error)
	GetProfile(ctx context.Context, userID string) (PlayerProfile, bool, error)
	ApplySettlement(ctx context.Context, gameID string, deltas []SettlementDelta) (bool, error)
}
