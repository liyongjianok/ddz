package profile

import (
	"time"

	"ddz/backend/internal/game"
)

// PlayerProfile 表示玩家在大厅侧的长期统计资料。
type PlayerProfile struct {
	UserID        string    `json:"user_id"`
	Level         int       `json:"level"`
	CoinBalance   int       `json:"coin_balance"`
	TotalGames    int       `json:"total_games"`
	Wins          int       `json:"wins"`
	LandlordGames int       `json:"landlord_games"`
	LandlordWins  int       `json:"landlord_wins"`
	FarmerGames   int       `json:"farmer_games"`
	FarmerWins    int       `json:"farmer_wins"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SettlementDelta 表示结算后单个玩家应写入统计的数据变化。
type SettlementDelta struct {
	UserID     string
	Role       game.Role
	DeltaScore int
	IsWinner   bool
}
