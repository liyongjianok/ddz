package room

import (
	"testing"

	"ddz/backend/internal/game"
)

func TestRobotHandStrengthScoresHighCardsBombsAndRocket(t *testing.T) {
	hand := mustCardsForRoom([]string{
		"BJ", "RJ",
		"S2", "H2",
		"SA", "HK",
		"S7", "H7", "D7", "C7",
		"S3", "H4", "D5", "C6", "S8",
	})

	strength := robotHandStrength(hand)
	if strength < 20 {
		t.Fatalf("strength = %d, want at least 20", strength)
	}
}

func TestChooseRobotBidUsesStrengthAndHighestBid(t *testing.T) {
	currentGame := newRobotBiddingGame()
	currentGame.Players[0].Hand = mustCardsForRoom([]string{
		"BJ", "RJ",
		"S2", "H2",
		"SA", "HK",
		"S7", "H7", "D7", "C7",
		"S3", "H4", "D5", "C6", "S8", "H9", "DT",
	})

	if score := chooseRobotBid(currentGame, 0); score != 3 {
		t.Fatalf("score = %d, want 3", score)
	}

	currentGame.BiddingState.HighestBid = 3
	if score := chooseRobotBid(currentGame, 0); score != 0 {
		t.Fatalf("score = %d, want 0 when highest bid is already 3", score)
	}
}

func TestChooseRobotBidPassesWeakHand(t *testing.T) {
	currentGame := newRobotBiddingGame()
	currentGame.Players[0].Hand = mustCardsForRoom([]string{
		"S3", "H4", "D5", "C6", "S8", "H9", "DT", "CJ", "SQ", "H6",
		"D7", "C8", "S9", "HT", "DJ", "CQ", "S5",
	})

	if score := chooseRobotBid(currentGame, 0); score != 0 {
		t.Fatalf("score = %d, want 0", score)
	}
}

func TestChooseRobotLeadPlaysWinningMove(t *testing.T) {
	currentGame := newRobotPlayingGame()
	currentGame.CurrentSeatIndex = 0
	currentGame.Players[0].Hand = mustCardsForRoom([]string{"S3", "H3"})
	currentGame.Players[0].RemainingCount = 2

	choice := chooseRobotPlay(currentGame, 0)
	if choice.pass {
		t.Fatal("choice should play")
	}
	if len(choice.cards) != 2 {
		t.Fatalf("card len = %d, want 2", len(choice.cards))
	}
	group, err := game.Recognize(choice.cards)
	if err != nil {
		t.Fatalf("Recognize() error = %v", err)
	}
	if group.Type != game.CardGroupTypePair {
		t.Fatalf("group type = %q, want pair", group.Type)
	}
}

func TestChooseRobotLeadAvoidsSingleWhenNextPlayerHasOneCard(t *testing.T) {
	currentGame := newRobotPlayingGame()
	currentGame.CurrentSeatIndex = 0
	currentGame.Players[0].Hand = mustCardsForRoom([]string{"S3", "H3", "S4"})
	currentGame.Players[0].RemainingCount = 3
	currentGame.Players[1].RemainingCount = 1

	choice := chooseRobotPlay(currentGame, 0)
	if choice.pass {
		t.Fatal("choice should play")
	}
	group, err := game.Recognize(choice.cards)
	if err != nil {
		t.Fatalf("Recognize() error = %v", err)
	}
	if group.Type == game.CardGroupTypeSingle {
		t.Fatalf("group type = %q, want non-single", group.Type)
	}
}

func TestChooseRobotResponsePassesForFarmerTeammate(t *testing.T) {
	currentGame := newRobotPlayingGame()
	currentGame.CurrentSeatIndex = 2
	currentGame.Players[0].Role = game.RoleLandlord
	currentGame.Players[1].Role = game.RoleFarmer
	currentGame.Players[2].Role = game.RoleFarmer
	currentGame.LastPlay = &game.Play{
		SeatIndex: 1,
		UserID:    "u2",
		Cards:     mustCardsForRoom([]string{"S3"}),
		Group:     mustRecognizeForRoom([]string{"S3"}),
	}
	currentGame.Players[2].Hand = mustCardsForRoom([]string{"S4", "H4", "D7"})
	currentGame.Players[2].RemainingCount = 3

	choice := chooseRobotPlay(currentGame, 2)
	if !choice.pass {
		t.Fatalf("choice should pass, got %v", cardCodesForRobotTest(choice.cards))
	}
}

func TestChooseRobotResponseUsesSmallestNonBomb(t *testing.T) {
	currentGame := newRobotPlayingGame()
	currentGame.CurrentSeatIndex = 1
	currentGame.LastPlay = &game.Play{
		SeatIndex: 0,
		UserID:    "u1",
		Cards:     mustCardsForRoom([]string{"S3"}),
		Group:     mustRecognizeForRoom([]string{"S3"}),
	}
	currentGame.Players[1].Hand = mustCardsForRoom([]string{"S4", "S5", "S8", "H8", "D8", "C8"})
	currentGame.Players[1].RemainingCount = 6

	choice := chooseRobotPlay(currentGame, 1)
	if choice.pass {
		t.Fatal("choice should play")
	}
	if len(choice.cards) != 1 || choice.cards[0].Code() != "S4" {
		t.Fatalf("cards = %v, want [S4]", cardCodesForRobotTest(choice.cards))
	}
}

func TestChooseRobotResponseDoesNotBombUnlessOpponentNearlyWins(t *testing.T) {
	currentGame := newRobotPlayingGame()
	currentGame.CurrentSeatIndex = 1
	currentGame.LastPlay = &game.Play{
		SeatIndex: 0,
		UserID:    "u1",
		Cards:     mustCardsForRoom([]string{"S9", "H9"}),
		Group:     mustRecognizeForRoom([]string{"S9", "H9"}),
	}
	currentGame.Players[1].Hand = mustCardsForRoom([]string{"S4", "H4", "D4", "C4", "S5"})
	currentGame.Players[1].RemainingCount = 5

	choice := chooseRobotPlay(currentGame, 1)
	if !choice.pass {
		t.Fatalf("choice should pass to preserve bomb, got %v", cardCodesForRobotTest(choice.cards))
	}

	currentGame.Players[0].RemainingCount = 1
	choice = chooseRobotPlay(currentGame, 1)
	if choice.pass {
		t.Fatal("choice should bomb when opponent is nearly out")
	}
	group, err := game.Recognize(choice.cards)
	if err != nil {
		t.Fatalf("Recognize() error = %v", err)
	}
	if group.Type != game.CardGroupTypeBomb {
		t.Fatalf("group type = %q, want bomb", group.Type)
	}
}

func TestHandleRobotTurnCanBidAboveZero(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "robot_a"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom u2 error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u3"}); err != nil {
		t.Fatalf("JoinRoom u3 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "robot_a", Ready: true}); err != nil {
		t.Fatalf("Ready robot error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u2", Ready: true}); err != nil {
		t.Fatalf("Ready u2 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u3", Ready: true}); err != nil {
		t.Fatalf("Ready u3 error = %v", err)
	}

	actor := manager.rooms[room.ID]
	actor.room.Seats[0].IsRobot = true
	actor.room.CurrentGame.Players[0].IsRobot = true
	actor.room.CurrentGame.CurrentSeatIndex = 0
	actor.room.CurrentGame.BiddingState.CurrentSeatIndex = 0
	actor.room.CurrentGame.Players[0].Hand = mustCardsForRoom([]string{
		"BJ", "RJ",
		"S2", "H2",
		"S7", "H7", "D7", "C7",
		"SA", "HK", "S3", "H4", "D5", "C6", "S8", "H9", "DT",
	})

	updatedRoom, action, err := manager.HandleRobotTurn(room.ID)
	if err != nil {
		t.Fatalf("HandleRobotTurn() error = %v", err)
	}
	if action != TimeoutActionAutoBid {
		t.Fatalf("action = %q, want %q", action, TimeoutActionAutoBid)
	}
	if got := updatedRoom.CurrentGame.BiddingState.Bids[0].Score; got != 3 {
		t.Fatalf("bid score = %d, want 3", got)
	}
}

func newRobotBiddingGame() *game.Game {
	return &game.Game{
		ID:               "g_robot_bid",
		Phase:            game.GamePhaseBidding,
		CurrentSeatIndex: 0,
		BiddingState: game.BiddingState{
			CurrentSeatIndex:    0,
			HighestBidSeatIndex: -1,
		},
		Players: []game.PlayerState{
			{UserID: "u1", SeatIndex: 0, Status: game.PlayerStatusPlaying, IsRobot: true},
			{UserID: "u2", SeatIndex: 1, Status: game.PlayerStatusPlaying},
			{UserID: "u3", SeatIndex: 2, Status: game.PlayerStatusPlaying},
		},
	}
}

func newRobotPlayingGame() *game.Game {
	return &game.Game{
		ID:                "g_robot_play",
		Phase:             game.GamePhasePlaying,
		LandlordSeatIndex: 0,
		CurrentSeatIndex:  0,
		Players: []game.PlayerState{
			{UserID: "u1", SeatIndex: 0, Role: game.RoleLandlord, Status: game.PlayerStatusPlaying, IsRobot: true, RemainingCount: 5},
			{UserID: "u2", SeatIndex: 1, Role: game.RoleFarmer, Status: game.PlayerStatusPlaying, IsRobot: true, RemainingCount: 5},
			{UserID: "u3", SeatIndex: 2, Role: game.RoleFarmer, Status: game.PlayerStatusPlaying, IsRobot: true, RemainingCount: 5},
		},
		Multiplier: 1,
	}
}

func cardCodesForRobotTest(cards []game.Card) []string {
	codes := make([]string, 0, len(cards))
	for _, card := range cards {
		codes = append(codes, card.Code())
	}
	return codes
}
