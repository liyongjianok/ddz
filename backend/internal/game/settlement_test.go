package game

import (
	"errors"
	"testing"
)

func TestSettleLandlordWins(t *testing.T) {
	game := newEndedGameForSettlement()
	game.Multiplier = 3
	game.Players[0].RemainingCount = 0
	game.Players[0].Hand = nil

	result, err := game.Settle(2)
	if err != nil {
		t.Fatalf("Settle() error = %v", err)
	}

	if result.WinnerSide != WinnerSideLandlord {
		t.Fatalf("winner side = %q, want %q", result.WinnerSide, WinnerSideLandlord)
	}
	if result.FinisherSeatIndex != 0 {
		t.Fatalf("finisher seat = %d, want 0", result.FinisherSeatIndex)
	}
	if result.BaseScore != 2 {
		t.Fatalf("base score = %d, want 2", result.BaseScore)
	}
	if result.Multiplier != 3 {
		t.Fatalf("multiplier = %d, want 3", result.Multiplier)
	}
	if result.UnitScore != 6 {
		t.Fatalf("unit score = %d, want 6", result.UnitScore)
	}

	assertSettlementPlayer(t, result.Players[0], 0, RoleLandlord, 12, true)
	assertSettlementPlayer(t, result.Players[1], 1, RoleFarmer, -6, false)
	assertSettlementPlayer(t, result.Players[2], 2, RoleFarmer, -6, false)

	if sumSettlement(result) != 0 {
		t.Fatalf("settlement total = %d, want 0", sumSettlement(result))
	}
}

func TestSettleFarmersWin(t *testing.T) {
	game := newEndedGameForSettlement()
	game.Multiplier = 2
	game.Players[1].RemainingCount = 0
	game.Players[1].Hand = nil

	result, err := game.Settle(3)
	if err != nil {
		t.Fatalf("Settle() error = %v", err)
	}

	if result.WinnerSide != WinnerSideFarmers {
		t.Fatalf("winner side = %q, want %q", result.WinnerSide, WinnerSideFarmers)
	}
	if result.FinisherSeatIndex != 1 {
		t.Fatalf("finisher seat = %d, want 1", result.FinisherSeatIndex)
	}
	if result.UnitScore != 6 {
		t.Fatalf("unit score = %d, want 6", result.UnitScore)
	}

	assertSettlementPlayer(t, result.Players[0], 0, RoleLandlord, -12, false)
	assertSettlementPlayer(t, result.Players[1], 1, RoleFarmer, 6, true)
	assertSettlementPlayer(t, result.Players[2], 2, RoleFarmer, 6, true)

	if sumSettlement(result) != 0 {
		t.Fatalf("settlement total = %d, want 0", sumSettlement(result))
	}
}

func TestSettleRejectsGameNotEnded(t *testing.T) {
	game := newManualPlayingGame()

	_, err := game.Settle(1)
	if !errors.Is(err, ErrGameNotEnded) {
		t.Fatalf("Settle() error = %v, want ErrGameNotEnded", err)
	}
}

func TestSettleRejectsInvalidBaseScore(t *testing.T) {
	game := newEndedGameForSettlement()
	game.Players[0].RemainingCount = 0

	_, err := game.Settle(0)
	if !errors.Is(err, ErrInvalidBaseScore) {
		t.Fatalf("Settle() error = %v, want ErrInvalidBaseScore", err)
	}
}

func TestSettleRejectsUndeterminedWinner(t *testing.T) {
	game := newEndedGameForSettlement()

	_, err := game.Settle(1)
	if !errors.Is(err, ErrWinnerUndetermined) {
		t.Fatalf("Settle() error = %v, want ErrWinnerUndetermined", err)
	}
}

func TestSettleReturnsCachedResult(t *testing.T) {
	game := newEndedGameForSettlement()
	game.Multiplier = 2
	game.Players[0].RemainingCount = 0

	first, err := game.Settle(2)
	if err != nil {
		t.Fatalf("first Settle() error = %v", err)
	}
	second, err := game.Settle(2)
	if err != nil {
		t.Fatalf("second Settle() error = %v", err)
	}

	if first != second {
		t.Fatal("Settle() should return cached result pointer")
	}
	if game.Settlement != first {
		t.Fatal("game settlement should point to cached result")
	}
}

func newEndedGameForSettlement() *Game {
	game := newManualPlayingGame()
	game.Phase = GamePhaseEnded
	game.CurrentSeatIndex = 0
	game.EndedAt = game.StartedAt
	return game
}

func assertSettlementPlayer(t *testing.T, player SettlementPlayer, seatIndex int, role Role, deltaScore int, isWinner bool) {
	t.Helper()
	if player.SeatIndex != seatIndex {
		t.Fatalf("seat index = %d, want %d", player.SeatIndex, seatIndex)
	}
	if player.Role != role {
		t.Fatalf("role = %q, want %q", player.Role, role)
	}
	if player.DeltaScore != deltaScore {
		t.Fatalf("delta score = %d, want %d", player.DeltaScore, deltaScore)
	}
	if player.IsWinner != isWinner {
		t.Fatalf("is winner = %t, want %t", player.IsWinner, isWinner)
	}
}

func sumSettlement(result *SettlementResult) int {
	total := 0
	for _, player := range result.Players {
		total += player.DeltaScore
	}
	return total
}
