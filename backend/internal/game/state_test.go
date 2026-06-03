package game

import (
	"errors"
	"math/rand"
	"testing"
)

func TestNewGameInitializesState(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	game, err := NewGame("g_001", defaultPlayers(), rng)
	if err != nil {
		t.Fatalf("NewGame() error = %v", err)
	}

	if game.ID != "g_001" {
		t.Fatalf("game.ID = %q, want %q", game.ID, "g_001")
	}
	if game.Phase != GamePhaseBidding {
		t.Fatalf("game.Phase = %q, want %q", game.Phase, GamePhaseBidding)
	}
	if len(game.Deck) != DeckCardCount {
		t.Fatalf("len(game.Deck) = %d, want %d", len(game.Deck), DeckCardCount)
	}
	if len(game.BottomCards) != BottomCardCount {
		t.Fatalf("len(game.BottomCards) = %d, want %d", len(game.BottomCards), BottomCardCount)
	}
	if len(game.Players) != PlayerCount {
		t.Fatalf("len(game.Players) = %d, want %d", len(game.Players), PlayerCount)
	}
	if game.CurrentSeatIndex != game.BiddingState.StartSeatIndex {
		t.Fatalf("current seat = %d, start seat = %d", game.CurrentSeatIndex, game.BiddingState.StartSeatIndex)
	}
	if game.CurrentSeatIndex < 0 || game.CurrentSeatIndex >= PlayerCount {
		t.Fatalf("current seat = %d, want within [0,%d)", game.CurrentSeatIndex, PlayerCount)
	}
	if game.BiddingState.HighestBid != 0 {
		t.Fatalf("highest bid = %d, want 0", game.BiddingState.HighestBid)
	}
	if game.BiddingState.HighestBidSeatIndex != -1 {
		t.Fatalf("highest bid seat = %d, want -1", game.BiddingState.HighestBidSeatIndex)
	}
	if game.LandlordSeatIndex != -1 {
		t.Fatalf("landlord seat = %d, want -1", game.LandlordSeatIndex)
	}
	if game.Multiplier != 1 {
		t.Fatalf("multiplier = %d, want 1", game.Multiplier)
	}
	if game.StartedAt.IsZero() {
		t.Fatal("started_at should be set")
	}

	seen := make(map[string]struct{}, DeckCardCount)
	for seatIndex, player := range game.Players {
		if player.SeatIndex != seatIndex {
			t.Fatalf("player seat index = %d, want %d", player.SeatIndex, seatIndex)
		}
		if player.Status != PlayerStatusPlaying {
			t.Fatalf("player[%d] status = %q, want %q", seatIndex, player.Status, PlayerStatusPlaying)
		}
		if player.Role != RoleNone {
			t.Fatalf("player[%d] role = %q, want empty", seatIndex, player.Role)
		}
		if len(player.Hand) != HandCardCount {
			t.Fatalf("len(player[%d].Hand) = %d, want %d", seatIndex, len(player.Hand), HandCardCount)
		}
		if player.RemainingCount != HandCardCount {
			t.Fatalf("player[%d].RemainingCount = %d, want %d", seatIndex, player.RemainingCount, HandCardCount)
		}
		for _, card := range player.Hand {
			code := card.Code()
			if _, exists := seen[code]; exists {
				t.Fatalf("duplicate dealt card %q", code)
			}
			seen[code] = struct{}{}
		}
	}
	for _, card := range game.BottomCards {
		code := card.Code()
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate bottom card %q", code)
		}
		seen[code] = struct{}{}
	}
	if len(seen) != DeckCardCount {
		t.Fatalf("unique card count = %d, want %d", len(seen), DeckCardCount)
	}
}

func TestNewGameRejectsInvalidSetup(t *testing.T) {
	testCases := []struct {
		name    string
		gameID  string
		players []GamePlayerInput
	}{
		{
			name:    "empty_game_id",
			gameID:  "",
			players: defaultPlayers(),
		},
		{
			name:   "too_few_players",
			gameID: "g_001",
			players: []GamePlayerInput{
				{UserID: "u1", SeatIndex: 0},
				{UserID: "u2", SeatIndex: 1},
			},
		},
		{
			name:   "duplicate_seat",
			gameID: "g_001",
			players: []GamePlayerInput{
				{UserID: "u1", SeatIndex: 0},
				{UserID: "u2", SeatIndex: 0},
				{UserID: "u3", SeatIndex: 2},
			},
		},
		{
			name:   "duplicate_user",
			gameID: "g_001",
			players: []GamePlayerInput{
				{UserID: "u1", SeatIndex: 0},
				{UserID: "u1", SeatIndex: 1},
				{UserID: "u3", SeatIndex: 2},
			},
		},
		{
			name:   "invalid_seat",
			gameID: "g_001",
			players: []GamePlayerInput{
				{UserID: "u1", SeatIndex: 0},
				{UserID: "u2", SeatIndex: 1},
				{UserID: "u3", SeatIndex: 3},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewGame(tc.gameID, tc.players, rand.New(rand.NewSource(1)))
			if !errors.Is(err, ErrInvalidGameSetup) {
				t.Fatalf("NewGame() error = %v, want ErrInvalidGameSetup", err)
			}
		})
	}
}

func defaultPlayers() []GamePlayerInput {
	return []GamePlayerInput{
		{UserID: "u1", SeatIndex: 0},
		{UserID: "u2", SeatIndex: 1},
		{UserID: "u3", SeatIndex: 2},
	}
}
