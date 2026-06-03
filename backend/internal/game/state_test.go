package game

import (
	"errors"
	"testing"
)

func TestNewGameInitializesState(t *testing.T) {
	rng := &fixedRNG{values: []int{1}}
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
	if game.BiddingState.RedealCount != 0 {
		t.Fatalf("redeal count = %d, want 0", game.BiddingState.RedealCount)
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
			_, err := NewGame(tc.gameID, tc.players, &fixedRNG{values: []int{0}})
			if !errors.Is(err, ErrInvalidGameSetup) {
				t.Fatalf("NewGame() error = %v, want ErrInvalidGameSetup", err)
			}
		})
	}
}

func TestPlaceBidRejectsOutOfTurn(t *testing.T) {
	game, _ := newBiddingGame(t, []int{0})

	err := game.PlaceBid(1, 1, &fixedRNG{values: []int{0}})
	if !errors.Is(err, ErrNotPlayerTurn) {
		t.Fatalf("PlaceBid() error = %v, want ErrNotPlayerTurn", err)
	}
}

func TestPlaceBidRejectsLowerOrEqualHighestBid(t *testing.T) {
	game, _ := newBiddingGame(t, []int{0})

	if err := game.PlaceBid(0, 1, &fixedRNG{values: []int{0}}); err != nil {
		t.Fatalf("first PlaceBid() error = %v", err)
	}
	err := game.PlaceBid(1, 1, &fixedRNG{values: []int{0}})
	if !errors.Is(err, ErrInvalidBid) {
		t.Fatalf("second PlaceBid() error = %v, want ErrInvalidBid", err)
	}
}

func TestPlaceBidThreeImmediatelyAssignsLandlord(t *testing.T) {
	game, _ := newBiddingGame(t, []int{0})

	if err := game.PlaceBid(0, 3, &fixedRNG{values: []int{0}}); err != nil {
		t.Fatalf("PlaceBid() error = %v", err)
	}

	if game.Phase != GamePhasePlaying {
		t.Fatalf("phase = %q, want %q", game.Phase, GamePhasePlaying)
	}
	if game.LandlordSeatIndex != 0 {
		t.Fatalf("landlord seat = %d, want 0", game.LandlordSeatIndex)
	}
	if game.CurrentSeatIndex != 0 {
		t.Fatalf("current seat = %d, want 0", game.CurrentSeatIndex)
	}
	if game.Multiplier != 3 {
		t.Fatalf("multiplier = %d, want 3", game.Multiplier)
	}
	if game.Players[0].Role != RoleLandlord {
		t.Fatalf("player[0].role = %q, want %q", game.Players[0].Role, RoleLandlord)
	}
	if len(game.Players[0].Hand) != 20 {
		t.Fatalf("len(player[0].Hand) = %d, want 20", len(game.Players[0].Hand))
	}
	for i := 1; i < PlayerCount; i++ {
		if game.Players[i].Role != RoleFarmer {
			t.Fatalf("player[%d].role = %q, want %q", i, game.Players[i].Role, RoleFarmer)
		}
	}
}

func TestPlaceBidAllPassRedealsOnce(t *testing.T) {
	game, rng := newBiddingGame(t, []int{0, 1})

	if err := game.PlaceBid(0, 0, rng); err != nil {
		t.Fatalf("PlaceBid 1 error = %v", err)
	}
	if err := game.PlaceBid(1, 0, rng); err != nil {
		t.Fatalf("PlaceBid 2 error = %v", err)
	}
	if err := game.PlaceBid(2, 0, rng); err != nil {
		t.Fatalf("PlaceBid 3 error = %v", err)
	}

	if game.Phase != GamePhaseBidding {
		t.Fatalf("phase = %q, want %q", game.Phase, GamePhaseBidding)
	}
	if game.BiddingState.RedealCount != 1 {
		t.Fatalf("redeal count = %d, want 1", game.BiddingState.RedealCount)
	}
	if game.BiddingState.HighestBid != 0 || game.BiddingState.HighestBidSeatIndex != -1 {
		t.Fatalf("highest bid state not reset: %+v", game.BiddingState)
	}
	if len(game.BiddingState.Bids) != 0 {
		t.Fatalf("bids len = %d, want 0", len(game.BiddingState.Bids))
	}
	if game.CurrentSeatIndex < 0 || game.CurrentSeatIndex >= PlayerCount {
		t.Fatalf("current seat = %d, want within [0,%d)", game.CurrentSeatIndex, PlayerCount)
	}
	for i := range game.Players {
		if len(game.Players[i].Hand) != HandCardCount {
			t.Fatalf("len(player[%d].Hand) = %d, want %d", i, len(game.Players[i].Hand), HandCardCount)
		}
		if game.Players[i].BidScore != 0 {
			t.Fatalf("player[%d].BidScore = %d, want 0", i, game.Players[i].BidScore)
		}
	}
}

func TestPlaceBidSecondAllPassAssignsRandomLandlord(t *testing.T) {
	game, rng := newBiddingGame(t, []int{0, 1, 2})

	if err := game.PlaceBid(0, 0, rng); err != nil {
		t.Fatalf("round1 bid1 error = %v", err)
	}
	if err := game.PlaceBid(1, 0, rng); err != nil {
		t.Fatalf("round1 bid2 error = %v", err)
	}
	if err := game.PlaceBid(2, 0, rng); err != nil {
		t.Fatalf("round1 bid3 error = %v", err)
	}

	startSeat := game.CurrentSeatIndex
	secondSeat := (startSeat + 1) % PlayerCount
	thirdSeat := (startSeat + 2) % PlayerCount

	if err := game.PlaceBid(startSeat, 0, rng); err != nil {
		t.Fatalf("round2 bid1 error = %v", err)
	}
	if err := game.PlaceBid(secondSeat, 0, rng); err != nil {
		t.Fatalf("round2 bid2 error = %v", err)
	}
	if err := game.PlaceBid(thirdSeat, 0, rng); err != nil {
		t.Fatalf("round2 bid3 error = %v", err)
	}

	if game.Phase != GamePhasePlaying {
		t.Fatalf("phase = %q, want %q", game.Phase, GamePhasePlaying)
	}
	if game.LandlordSeatIndex < 0 || game.LandlordSeatIndex >= PlayerCount {
		t.Fatalf("landlord seat = %d, want within [0,%d)", game.LandlordSeatIndex, PlayerCount)
	}
	if game.Multiplier != 1 {
		t.Fatalf("multiplier = %d, want 1", game.Multiplier)
	}
	if len(game.Players[game.LandlordSeatIndex].Hand) != 20 {
		t.Fatalf("len(player[%d].Hand) = %d, want 20", game.LandlordSeatIndex, len(game.Players[game.LandlordSeatIndex].Hand))
	}
	if game.Players[game.LandlordSeatIndex].Role != RoleLandlord {
		t.Fatalf("landlord role = %q, want %q", game.Players[game.LandlordSeatIndex].Role, RoleLandlord)
	}
}

func TestPlayCardsRejectsOutOfTurn(t *testing.T) {
	game := newManualPlayingGame()

	err := game.PlayCards(1, mustCardsForGame([]string{"S4"}))
	if !errors.Is(err, ErrNotPlayerTurn) {
		t.Fatalf("PlayCards() error = %v, want ErrNotPlayerTurn", err)
	}
}

func TestPlayCardsRejectsCardsNotInHand(t *testing.T) {
	game := newManualPlayingGame()

	err := game.PlayCards(0, mustCardsForGame([]string{"RJ"}))
	if !errors.Is(err, ErrInvalidCardSet) {
		t.Fatalf("PlayCards() error = %v, want ErrInvalidCardSet", err)
	}
}

func TestPassRejectsOnFreshTrick(t *testing.T) {
	game := newManualPlayingGame()

	err := game.Pass(0)
	if !errors.Is(err, ErrCannotPass) {
		t.Fatalf("Pass() error = %v, want ErrCannotPass", err)
	}
}

func TestPlayCardsRemovesCardsAndAdvancesTurn(t *testing.T) {
	game := newManualPlayingGame()

	if err := game.PlayCards(0, mustCardsForGame([]string{"S3"})); err != nil {
		t.Fatalf("PlayCards() error = %v", err)
	}

	if game.CurrentSeatIndex != 1 {
		t.Fatalf("current seat = %d, want 1", game.CurrentSeatIndex)
	}
	if game.PassCount != 0 {
		t.Fatalf("pass count = %d, want 0", game.PassCount)
	}
	if game.LastPlay == nil {
		t.Fatal("last play should be set")
	}
	if len(game.Players[0].Hand) != 3 {
		t.Fatalf("len(player[0].Hand) = %d, want 3", len(game.Players[0].Hand))
	}
	if game.Players[0].RemainingCount != 3 {
		t.Fatalf("remaining count = %d, want 3", game.Players[0].RemainingCount)
	}
}

func TestPlayCardsRequiresBeatingPreviousPlay(t *testing.T) {
	game := newManualPlayingGame()
	if err := game.PlayCards(0, mustCardsForGame([]string{"S3"})); err != nil {
		t.Fatalf("lead PlayCards() error = %v", err)
	}

	err := game.PlayCards(1, mustCardsForGame([]string{"S4", "H4"}))
	if !errors.Is(err, ErrInvalidCardSet) {
		t.Fatalf("PlayCards() error = %v, want ErrInvalidCardSet", err)
	}

	if err := game.PlayCards(1, mustCardsForGame([]string{"S4"})); err != nil {
		t.Fatalf("response PlayCards() error = %v", err)
	}
	if game.CurrentSeatIndex != 2 {
		t.Fatalf("current seat = %d, want 2", game.CurrentSeatIndex)
	}
}

func TestPassTwiceReturnsLeadToLastPlayer(t *testing.T) {
	game := newManualPlayingGame()
	if err := game.PlayCards(0, mustCardsForGame([]string{"S3"})); err != nil {
		t.Fatalf("lead PlayCards() error = %v", err)
	}

	if err := game.Pass(1); err != nil {
		t.Fatalf("first Pass() error = %v", err)
	}
	if game.CurrentSeatIndex != 2 {
		t.Fatalf("current seat after first pass = %d, want 2", game.CurrentSeatIndex)
	}
	if err := game.Pass(2); err != nil {
		t.Fatalf("second Pass() error = %v", err)
	}

	if game.CurrentSeatIndex != 0 {
		t.Fatalf("current seat after second pass = %d, want 0", game.CurrentSeatIndex)
	}
	if game.LastPlay != nil {
		t.Fatal("last play should be cleared after two passes")
	}
	if game.PassCount != 0 {
		t.Fatalf("pass count = %d, want 0", game.PassCount)
	}
}

func TestPlayCardsEndsGameOnEmptyHand(t *testing.T) {
	game := newManualPlayingGame()
	game.Players[0].Hand = mustCardsForGame([]string{"S3"})
	game.Players[0].RemainingCount = 1

	if err := game.PlayCards(0, mustCardsForGame([]string{"S3"})); err != nil {
		t.Fatalf("PlayCards() error = %v", err)
	}

	if game.Phase != GamePhaseEnded {
		t.Fatalf("phase = %q, want %q", game.Phase, GamePhaseEnded)
	}
	if game.EndedAt.IsZero() {
		t.Fatal("ended_at should be set")
	}
	if game.Players[0].RemainingCount != 0 {
		t.Fatalf("remaining count = %d, want 0", game.Players[0].RemainingCount)
	}
}

func defaultPlayers() []GamePlayerInput {
	return []GamePlayerInput{
		{UserID: "u1", SeatIndex: 0},
		{UserID: "u2", SeatIndex: 1},
		{UserID: "u3", SeatIndex: 2},
	}
}

func newManualPlayingGame() *Game {
	return &Game{
		ID:                "g_play",
		Phase:             GamePhasePlaying,
		LandlordSeatIndex: 0,
		CurrentSeatIndex:  0,
		BottomCards:       mustCardsForGame([]string{"S9", "H9", "D9"}),
		Players: []PlayerState{
			{
				UserID:         "u1",
				SeatIndex:      0,
				Role:           RoleLandlord,
				Status:         PlayerStatusPlaying,
				Hand:           mustCardsForGame([]string{"S3", "H3", "D3", "S5"}),
				RemainingCount: 4,
			},
			{
				UserID:         "u2",
				SeatIndex:      1,
				Role:           RoleFarmer,
				Status:         PlayerStatusPlaying,
				Hand:           mustCardsForGame([]string{"S4", "H4", "D4", "S6"}),
				RemainingCount: 4,
			},
			{
				UserID:         "u3",
				SeatIndex:      2,
				Role:           RoleFarmer,
				Status:         PlayerStatusPlaying,
				Hand:           mustCardsForGame([]string{"S7", "H7", "D7", "S8"}),
				RemainingCount: 4,
			},
		},
		Multiplier: 1,
	}
}

func newBiddingGame(t *testing.T, rngValues []int) (*Game, *fixedRNG) {
	t.Helper()
	rng := &fixedRNG{values: rngValues}
	game, err := NewGame("g_001", defaultPlayers(), rng)
	if err != nil {
		t.Fatalf("NewGame() error = %v", err)
	}
	return game, rng
}

type fixedRNG struct {
	values []int
	index  int
}

func (r *fixedRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	if n != PlayerCount {
		return 0
	}
	if len(r.values) == 0 {
		return 0
	}
	value := r.values[r.index%len(r.values)]
	r.index++
	value %= n
	if value < 0 {
		value += n
	}
	return value
}

func mustCardsForGame(codes []string) []Card {
	cards := make([]Card, 0, len(codes))
	for _, code := range codes {
		card, err := ParseCard(code)
		if err != nil {
			panic(err)
		}
		cards = append(cards, card)
	}
	return cards
}
