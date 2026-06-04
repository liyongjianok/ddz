package room

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ddz/backend/internal/game"
)

func TestGetRoomSnapshotWaitingRoom(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1", BaseScore: 2})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}

	snapshot, err := manager.GetRoomSnapshot(room.ID, "u1")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}

	if snapshot.Room.RoomID != room.ID {
		t.Fatalf("room id = %q, want %q", snapshot.Room.RoomID, room.ID)
	}
	if snapshot.Room.Status != string(RoomStatusWaiting) {
		t.Fatalf("room status = %q, want %q", snapshot.Room.Status, RoomStatusWaiting)
	}
	if snapshot.Room.BaseScore != 2 {
		t.Fatalf("base score = %d, want 2", snapshot.Room.BaseScore)
	}
	if snapshot.Game != nil {
		t.Fatal("waiting room snapshot should not contain game")
	}
	if len(snapshot.Me.Hand) != 0 {
		t.Fatalf("hand len = %d, want 0", len(snapshot.Me.Hand))
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("player len = %d, want 1", len(snapshot.Players))
	}
	if !snapshot.Players[0].Ready {
		t.Fatal("player should be ready in waiting room snapshot")
	}
}

func TestGetRoomSnapshotIncludesOwnHandAndHidesOthers(t *testing.T) {
	manager, roomID := newStartedRoomForSnapshot(t)

	snapshot, err := manager.GetRoomSnapshot(roomID, "u1")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}

	if snapshot.Game == nil {
		t.Fatal("started room snapshot should contain game")
	}
	if snapshot.Game.Phase != string(game.GamePhaseBidding) {
		t.Fatalf("game phase = %q, want %q", snapshot.Game.Phase, game.GamePhaseBidding)
	}
	if len(snapshot.Me.Hand) != game.HandCardCount {
		t.Fatalf("hand len = %d, want %d", len(snapshot.Me.Hand), game.HandCardCount)
	}
	if len(snapshot.Players) != game.PlayerCount {
		t.Fatalf("player len = %d, want %d", len(snapshot.Players), game.PlayerCount)
	}
	if snapshot.Game.BottomCards != nil {
		t.Fatal("bottom cards should stay hidden before landlord is decided")
	}

	actor := manager.rooms[roomID]
	hiddenCard := actor.room.CurrentGame.Players[1].Hand[0].Code()
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), `"`+hiddenCard+`"`) {
		t.Fatalf("snapshot leaked other player's hand card %q", hiddenCard)
	}
}

func TestGetRoomSnapshotIncludesSettlementAfterGameEnded(t *testing.T) {
	manager, roomID := newStartedRoomForSnapshot(t)
	actor := manager.rooms[roomID]
	currentGame := actor.room.CurrentGame

	currentGame.Phase = game.GamePhaseEnded
	currentGame.EndedAt = time.Now().UTC()
	currentGame.LandlordSeatIndex = 0
	currentGame.CurrentSeatIndex = 0
	currentGame.Multiplier = 3
	currentGame.Players[0].Role = game.RoleLandlord
	currentGame.Players[1].Role = game.RoleFarmer
	currentGame.Players[2].Role = game.RoleFarmer
	currentGame.Players[0].Hand = nil
	currentGame.Players[0].RemainingCount = 0
	currentGame.Players[1].Hand = currentGame.Players[1].Hand[:2]
	currentGame.Players[1].RemainingCount = 2
	currentGame.Players[2].Hand = currentGame.Players[2].Hand[:3]
	currentGame.Players[2].RemainingCount = 3

	snapshot, err := manager.GetRoomSnapshot(roomID, "u2")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}

	if snapshot.Game == nil {
		t.Fatal("ended room snapshot should contain game")
	}
	if snapshot.Game.Settlement == nil {
		t.Fatal("ended room snapshot should contain settlement")
	}
	if snapshot.Game.Settlement.WinnerSide != string(game.WinnerSideLandlord) {
		t.Fatalf("winner side = %q, want %q", snapshot.Game.Settlement.WinnerSide, game.WinnerSideLandlord)
	}
	if snapshot.Game.Settlement.FinalMultiplier != 3 {
		t.Fatalf("final multiplier = %d, want 3", snapshot.Game.Settlement.FinalMultiplier)
	}
	if len(snapshot.Game.BottomCards) != game.BottomCardCount {
		t.Fatalf("bottom card len = %d, want %d", len(snapshot.Game.BottomCards), game.BottomCardCount)
	}
	if len(snapshot.Me.Hand) != 2 {
		t.Fatalf("me hand len = %d, want 2", len(snapshot.Me.Hand))
	}
	if len(snapshot.Game.Settlement.Players) != game.PlayerCount {
		t.Fatalf("settlement player len = %d, want %d", len(snapshot.Game.Settlement.Players), game.PlayerCount)
	}
}

func TestGetRoomSnapshotRejectsUserNotInRoom(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	_, err = manager.GetRoomSnapshot(room.ID, "u9")
	if !errors.Is(err, ErrUserNotInRoom) {
		t.Fatalf("GetRoomSnapshot() error = %v, want ErrUserNotInRoom", err)
	}
}

func newStartedRoomForSnapshot(t *testing.T) (*Manager, string) {
	t.Helper()

	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1", BaseScore: 1})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom u2 error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u3"}); err != nil {
		t.Fatalf("JoinRoom u3 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready u1 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u2", Ready: true}); err != nil {
		t.Fatalf("Ready u2 error = %v", err)
	}
	if _, _, started, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u3", Ready: true}); err != nil {
		t.Fatalf("Ready u3 error = %v", err)
	} else if !started {
		t.Fatal("room should be started")
	}

	return manager, room.ID
}
