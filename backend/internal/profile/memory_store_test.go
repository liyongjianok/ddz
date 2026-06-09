package profile

import (
	"context"
	"testing"

	"ddz/backend/internal/game"
)

func TestMemoryStoreEnsureProfileUsesDefaults(t *testing.T) {
	store := NewMemoryStore()

	profile, err := store.EnsureProfile(context.Background(), "u1")
	if err != nil {
		t.Fatalf("EnsureProfile() error = %v", err)
	}
	if profile.Level != 1 {
		t.Fatalf("level = %d, want 1", profile.Level)
	}
	if profile.CoinBalance != 10000 {
		t.Fatalf("coin_balance = %d, want 10000", profile.CoinBalance)
	}
}

func TestMemoryStoreApplySettlementIsIdempotent(t *testing.T) {
	store := NewMemoryStore()

	applied, err := store.ApplySettlement(context.Background(), "g1", []SettlementDelta{
		{UserID: "u1", Role: game.RoleLandlord, DeltaScore: 6, IsWinner: true},
		{UserID: "u2", Role: game.RoleFarmer, DeltaScore: -3, IsWinner: false},
		{UserID: "u3", Role: game.RoleFarmer, DeltaScore: -3, IsWinner: false},
	})
	if err != nil {
		t.Fatalf("ApplySettlement() error = %v", err)
	}
	if !applied {
		t.Fatal("first apply should be true")
	}

	applied, err = store.ApplySettlement(context.Background(), "g1", []SettlementDelta{
		{UserID: "u1", Role: game.RoleLandlord, DeltaScore: 6, IsWinner: true},
	})
	if err != nil {
		t.Fatalf("ApplySettlement() second error = %v", err)
	}
	if applied {
		t.Fatal("second apply should be false")
	}

	profile, ok, err := store.GetProfile(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if !ok {
		t.Fatal("GetProfile() ok = false, want true")
	}
	if profile.TotalGames != 1 {
		t.Fatalf("total_games = %d, want 1", profile.TotalGames)
	}
	if profile.Wins != 1 {
		t.Fatalf("wins = %d, want 1", profile.Wins)
	}
	if profile.LandlordWins != 1 {
		t.Fatalf("landlord_wins = %d, want 1", profile.LandlordWins)
	}
	if profile.CoinBalance != 10006 {
		t.Fatalf("coin_balance = %d, want 10006", profile.CoinBalance)
	}
}
