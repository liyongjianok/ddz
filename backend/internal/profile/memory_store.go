package profile

import (
	"context"
	"sync"
	"time"

	"ddz/backend/internal/game"
)

// MemoryStore 是玩家资料的内存实现。
type MemoryStore struct {
	mu          sync.Mutex
	profiles    map[string]PlayerProfile
	appliedGame map[string]struct{}
	now         func() time.Time
}

// NewMemoryStore 创建内存版玩家资料存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		profiles:    make(map[string]PlayerProfile),
		appliedGame: make(map[string]struct{}),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *MemoryStore) EnsureProfile(_ context.Context, userID string) (PlayerProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ensureProfileLocked(userID), nil
}

func (s *MemoryStore) GetProfile(_ context.Context, userID string) (PlayerProfile, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerProfile, ok := s.profiles[userID]
	return playerProfile, ok, nil
}

func (s *MemoryStore) ApplySettlement(_ context.Context, gameID string, deltas []SettlementDelta) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.appliedGame[gameID]; ok {
		return false, nil
	}

	now := s.now()
	for _, delta := range deltas {
		playerProfile := s.ensureProfileLocked(delta.UserID)
		playerProfile.TotalGames++
		if delta.IsWinner {
			playerProfile.Wins++
		}
		playerProfile.CoinBalance += delta.DeltaScore
		switch delta.Role {
		case game.RoleLandlord:
			playerProfile.LandlordGames++
			if delta.IsWinner {
				playerProfile.LandlordWins++
			}
		case game.RoleFarmer:
			playerProfile.FarmerGames++
			if delta.IsWinner {
				playerProfile.FarmerWins++
			}
		}
		playerProfile.UpdatedAt = now
		s.profiles[delta.UserID] = playerProfile
	}

	s.appliedGame[gameID] = struct{}{}
	return true, nil
}

func (s *MemoryStore) ensureProfileLocked(userID string) PlayerProfile {
	playerProfile, ok := s.profiles[userID]
	if ok {
		return playerProfile
	}

	playerProfile = PlayerProfile{
		UserID:      userID,
		Level:       1,
		CoinBalance: 10000,
		UpdatedAt:   s.now(),
	}
	s.profiles[userID] = playerProfile
	return playerProfile
}
