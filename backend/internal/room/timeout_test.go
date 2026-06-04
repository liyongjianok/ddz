package room

import (
	"testing"
	"time"

	"ddz/backend/internal/game"
)

func TestHandleTimeoutAutoBidsZeroAndAdvancesTurn(t *testing.T) {
	manager, roomID := newStartedRoomForSnapshot(t)
	actor := manager.rooms[roomID]
	actor.room.DeadlineAt = time.Now().UTC().Add(-time.Second)

	currentSeat := actor.room.CurrentGame.CurrentSeatIndex

	room, action, err := manager.HandleTimeout(roomID)
	if err != nil {
		t.Fatalf("HandleTimeout() error = %v", err)
	}
	if action != TimeoutActionAutoBid {
		t.Fatalf("action = %q, want %q", action, TimeoutActionAutoBid)
	}
	if len(room.CurrentGame.BiddingState.Bids) != 1 {
		t.Fatalf("bid len = %d, want 1", len(room.CurrentGame.BiddingState.Bids))
	}

	bid := room.CurrentGame.BiddingState.Bids[0]
	if bid.SeatIndex != currentSeat {
		t.Fatalf("bid seat = %d, want %d", bid.SeatIndex, currentSeat)
	}
	if bid.Score != 0 {
		t.Fatalf("bid score = %d, want 0", bid.Score)
	}
	if room.CurrentGame.CurrentSeatIndex != (currentSeat+1)%game.PlayerCount {
		t.Fatalf("current seat = %d, want %d", room.CurrentGame.CurrentSeatIndex, (currentSeat+1)%game.PlayerCount)
	}
	if room.DeadlineAt.IsZero() {
		t.Fatal("deadline_at should be refreshed")
	}
	if !room.DeadlineAt.After(time.Now().UTC().Add(-time.Second)) {
		t.Fatalf("deadline_at = %v, want refreshed future time", room.DeadlineAt)
	}
}

func TestHandleTimeoutAutoPassWhenPassAllowed(t *testing.T) {
	manager, roomID := newStartedRoomForSnapshot(t)
	actor := manager.rooms[roomID]
	actor.room.Status = RoomStatusPlaying
	actor.room.CurrentGame = newTimeoutPassGame()
	actor.room.DeadlineAt = time.Now().UTC().Add(-time.Second)

	room, action, err := manager.HandleTimeout(roomID)
	if err != nil {
		t.Fatalf("HandleTimeout() error = %v", err)
	}
	if action != TimeoutActionAutoPass {
		t.Fatalf("action = %q, want %q", action, TimeoutActionAutoPass)
	}
	if room.CurrentGame.PassCount != 1 {
		t.Fatalf("pass count = %d, want 1", room.CurrentGame.PassCount)
	}
	if room.CurrentGame.CurrentSeatIndex != 2 {
		t.Fatalf("current seat = %d, want 2", room.CurrentGame.CurrentSeatIndex)
	}
	if room.CurrentGame.LastPlay == nil {
		t.Fatal("last play should be kept after a single pass")
	}
	if room.DeadlineAt.IsZero() {
		t.Fatal("deadline_at should be refreshed")
	}
}

func TestHandleTimeoutAutoPlaySmallestMoveWhenLeadTurn(t *testing.T) {
	manager, roomID := newStartedRoomForSnapshot(t)
	actor := manager.rooms[roomID]
	actor.room.Status = RoomStatusPlaying
	actor.room.CurrentGame = newTimeoutLeadGame()
	actor.room.DeadlineAt = time.Now().UTC().Add(-time.Second)

	room, action, err := manager.HandleTimeout(roomID)
	if err != nil {
		t.Fatalf("HandleTimeout() error = %v", err)
	}
	if action != TimeoutActionAutoPlay {
		t.Fatalf("action = %q, want %q", action, TimeoutActionAutoPlay)
	}
	if room.CurrentGame.LastPlay == nil {
		t.Fatal("last play should be created")
	}
	if len(room.CurrentGame.LastPlay.Cards) != 1 {
		t.Fatalf("last play card len = %d, want 1", len(room.CurrentGame.LastPlay.Cards))
	}
	if room.CurrentGame.LastPlay.Cards[0].Code() != "S3" {
		t.Fatalf("last play card = %q, want %q", room.CurrentGame.LastPlay.Cards[0].Code(), "S3")
	}
	if room.CurrentGame.CurrentSeatIndex != 1 {
		t.Fatalf("current seat = %d, want 1", room.CurrentGame.CurrentSeatIndex)
	}
	if room.CurrentGame.Players[0].RemainingCount != 3 {
		t.Fatalf("remaining count = %d, want 3", room.CurrentGame.Players[0].RemainingCount)
	}
	if room.DeadlineAt.IsZero() {
		t.Fatal("deadline_at should be refreshed")
	}
}

func TestHandleTimeoutDoesNotRepeatAfterManualAction(t *testing.T) {
	manager, roomID := newStartedRoomForSnapshot(t)
	actor := manager.rooms[roomID]
	actor.room.DeadlineAt = time.Now().UTC().Add(-time.Second)

	currentSeat := actor.room.CurrentGame.CurrentSeatIndex
	currentUserID := actor.room.CurrentGame.Players[currentSeat].UserID

	room, err := manager.Bid(BidInput{
		RoomID: roomID,
		UserID: currentUserID,
		Score:  1,
	})
	if err != nil {
		t.Fatalf("Bid() error = %v", err)
	}
	if len(room.CurrentGame.BiddingState.Bids) != 1 {
		t.Fatalf("bid len after manual action = %d, want 1", len(room.CurrentGame.BiddingState.Bids))
	}

	timeoutRoom, action, err := manager.HandleTimeout(roomID)
	if err != nil {
		t.Fatalf("HandleTimeout() error = %v", err)
	}
	if action != TimeoutActionNone {
		t.Fatalf("action = %q, want %q", action, TimeoutActionNone)
	}
	if len(timeoutRoom.CurrentGame.BiddingState.Bids) != 1 {
		t.Fatalf("bid len after timeout check = %d, want 1", len(timeoutRoom.CurrentGame.BiddingState.Bids))
	}
	if timeoutRoom.CurrentGame.BiddingState.Bids[0].Score != 1 {
		t.Fatalf("bid score after timeout check = %d, want 1", timeoutRoom.CurrentGame.BiddingState.Bids[0].Score)
	}
}

func TestGetRoomSnapshotIncludesDeadlineAtDuringActiveGame(t *testing.T) {
	manager, roomID := newStartedRoomForSnapshot(t)

	snapshot, err := manager.GetRoomSnapshot(roomID, "u1")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}
	if snapshot.Game == nil {
		t.Fatal("snapshot game should not be nil")
	}
	if snapshot.Game.DeadlineAt.IsZero() {
		t.Fatal("snapshot deadline_at should be set")
	}
}

func newTimeoutPassGame() *game.Game {
	group := mustRecognizeForRoom([]string{"S3"})
	return &game.Game{
		ID:                "g_timeout_pass",
		Phase:             game.GamePhasePlaying,
		LandlordSeatIndex: 0,
		CurrentSeatIndex:  1,
		BottomCards:       mustCardsForRoom([]string{"S9", "H9", "D9"}),
		LastPlay: &game.Play{
			SeatIndex: 0,
			UserID:    "u1",
			Cards:     mustCardsForRoom([]string{"S3"}),
			Group:     group,
			CreatedAt: time.Now().UTC().Add(-2 * time.Second),
		},
		Players: []game.PlayerState{
			{
				UserID:         "u1",
				SeatIndex:      0,
				Role:           game.RoleLandlord,
				Status:         game.PlayerStatusPlaying,
				Hand:           mustCardsForRoom([]string{"H3", "D3", "S5"}),
				RemainingCount: 3,
			},
			{
				UserID:         "u2",
				SeatIndex:      1,
				Role:           game.RoleFarmer,
				Status:         game.PlayerStatusPlaying,
				Hand:           mustCardsForRoom([]string{"S4", "H4", "D4", "S6"}),
				RemainingCount: 4,
			},
			{
				UserID:         "u3",
				SeatIndex:      2,
				Role:           game.RoleFarmer,
				Status:         game.PlayerStatusPlaying,
				Hand:           mustCardsForRoom([]string{"S7", "H7", "D7", "S8"}),
				RemainingCount: 4,
			},
		},
		Multiplier: 1,
	}
}

func newTimeoutLeadGame() *game.Game {
	return &game.Game{
		ID:                "g_timeout_lead",
		Phase:             game.GamePhasePlaying,
		LandlordSeatIndex: 0,
		CurrentSeatIndex:  0,
		BottomCards:       mustCardsForRoom([]string{"S9", "H9", "D9"}),
		Players: []game.PlayerState{
			{
				UserID:         "u1",
				SeatIndex:      0,
				Role:           game.RoleLandlord,
				Status:         game.PlayerStatusPlaying,
				Hand:           mustCardsForRoom([]string{"S3", "S5", "H7", "D9"}),
				RemainingCount: 4,
			},
			{
				UserID:         "u2",
				SeatIndex:      1,
				Role:           game.RoleFarmer,
				Status:         game.PlayerStatusPlaying,
				Hand:           mustCardsForRoom([]string{"S4", "H4", "D4", "S6"}),
				RemainingCount: 4,
			},
			{
				UserID:         "u3",
				SeatIndex:      2,
				Role:           game.RoleFarmer,
				Status:         game.PlayerStatusPlaying,
				Hand:           mustCardsForRoom([]string{"S7", "H7", "D7", "S8"}),
				RemainingCount: 4,
			},
		},
		Multiplier: 1,
	}
}

func mustCardsForRoom(codes []string) []game.Card {
	cards := make([]game.Card, 0, len(codes))
	for _, code := range codes {
		card, err := game.ParseCard(code)
		if err != nil {
			panic(err)
		}
		cards = append(cards, card)
	}
	return cards
}

func mustRecognizeForRoom(codes []string) game.CardGroup {
	group, err := game.Recognize(mustCardsForRoom(codes))
	if err != nil {
		panic(err)
	}
	return group
}
