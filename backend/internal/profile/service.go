package profile

import (
	"context"

	"ddz/backend/internal/room"
)

// Service 封装玩家资料和统计更新逻辑。
type Service struct {
	store Store
}

// NewService 创建玩家资料服务。
func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{store: store}
}

// EnsureProfile 确保玩家资料存在。
func (s *Service) EnsureProfile(ctx context.Context, userID string) (PlayerProfile, error) {
	return s.store.EnsureProfile(ctx, userID)
}

// GetProfile 获取玩家资料，不存在时自动初始化默认资料。
func (s *Service) GetProfile(ctx context.Context, userID string) (PlayerProfile, error) {
	playerProfile, ok, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return PlayerProfile{}, err
	}
	if ok {
		return playerProfile, nil
	}
	return s.store.EnsureProfile(ctx, userID)
}

// ApplySettlementFromRoom 根据房间结算结果更新玩家统计。
func (s *Service) ApplySettlementFromRoom(ctx context.Context, currentRoom *room.Room) error {
	if currentRoom == nil || currentRoom.CurrentGame == nil || currentRoom.CurrentGame.Settlement == nil {
		return nil
	}

	deltas := make([]SettlementDelta, 0, len(currentRoom.CurrentGame.Settlement.Players))
	for _, player := range currentRoom.CurrentGame.Settlement.Players {
		deltas = append(deltas, SettlementDelta{
			UserID:     player.UserID,
			Role:       player.Role,
			DeltaScore: player.DeltaScore,
			IsWinner:   player.IsWinner,
		})
	}

	_, err := s.store.ApplySettlement(ctx, currentRoom.CurrentGame.ID, deltas)
	return err
}
