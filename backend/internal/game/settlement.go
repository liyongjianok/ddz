package game

import (
	"errors"
	"time"
)

var ErrGameNotEnded = errors.New("game not ended")
var ErrInvalidBaseScore = errors.New("invalid base score")
var ErrWinnerUndetermined = errors.New("winner undetermined")

type WinnerSide string

const (
	WinnerSideNone     WinnerSide = ""
	WinnerSideLandlord WinnerSide = "landlord"
	WinnerSideFarmers  WinnerSide = "farmers"
)

// SettlementPlayer 表示单个玩家在本局结算中的分数变化。
type SettlementPlayer struct {
	UserID         string
	SeatIndex      int
	Role           Role
	RemainingCount int
	DeltaScore     int
	IsWinner       bool
}

// SettlementResult 表示整局游戏的结算结果。
type SettlementResult struct {
	BaseScore         int
	Multiplier        int
	UnitScore         int
	WinnerSide        WinnerSide
	FinisherSeatIndex int
	Players           []SettlementPlayer
	SettledAt         time.Time
}

// Settle 根据胜负方、底分和倍数计算当前对局的结算结果。
func (g *Game) Settle(baseScore int) (*SettlementResult, error) {
	if g.Phase != GamePhaseEnded {
		return nil, ErrGameNotEnded
	}
	if baseScore <= 0 {
		return nil, ErrInvalidBaseScore
	}
	if g.Settlement != nil {
		if g.Settlement.BaseScore != baseScore {
			return nil, ErrInvalidBaseScore
		}
		return g.Settlement, nil
	}

	winnerSide, finisherSeatIndex, err := g.determineWinnerSide()
	if err != nil {
		return nil, err
	}

	unitScore := baseScore * g.Multiplier
	result := &SettlementResult{
		BaseScore:         baseScore,
		Multiplier:        g.Multiplier,
		UnitScore:         unitScore,
		WinnerSide:        winnerSide,
		FinisherSeatIndex: finisherSeatIndex,
		Players:           make([]SettlementPlayer, 0, len(g.Players)),
		SettledAt:         time.Now().UTC(),
	}

	totalDelta := 0
	for _, player := range g.Players {
		delta, isWinner := settlementDelta(player.Role, winnerSide, unitScore)
		totalDelta += delta
		result.Players = append(result.Players, SettlementPlayer{
			UserID:         player.UserID,
			SeatIndex:      player.SeatIndex,
			Role:           player.Role,
			RemainingCount: player.RemainingCount,
			DeltaScore:     delta,
			IsWinner:       isWinner,
		})
	}

	if totalDelta != 0 {
		return nil, ErrWinnerUndetermined
	}

	g.Settlement = result
	return result, nil
}

// determineWinnerSide 根据出完手牌的一方判定地主或农民获胜。
func (g *Game) determineWinnerSide() (WinnerSide, int, error) {
	if g.LandlordSeatIndex < 0 || g.LandlordSeatIndex >= len(g.Players) {
		return WinnerSideNone, -1, ErrWinnerUndetermined
	}

	for _, player := range g.Players {
		if player.RemainingCount != 0 {
			continue
		}
		switch player.Role {
		case RoleLandlord:
			return WinnerSideLandlord, player.SeatIndex, nil
		case RoleFarmer:
			return WinnerSideFarmers, player.SeatIndex, nil
		}
	}

	return WinnerSideNone, -1, ErrWinnerUndetermined
}

func settlementDelta(role Role, winnerSide WinnerSide, unitScore int) (int, bool) {
	switch winnerSide {
	case WinnerSideLandlord:
		if role == RoleLandlord {
			return unitScore * 2, true
		}
		return -unitScore, false
	case WinnerSideFarmers:
		if role == RoleLandlord {
			return -unitScore * 2, false
		}
		return unitScore, true
	default:
		return 0, false
	}
}
