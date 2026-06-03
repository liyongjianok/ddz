package room

import (
	"errors"
	"testing"
)

func TestCreateRoomAssignsCreatorSeat(t *testing.T) {
	manager := NewManager()
	preferredSeat := 2

	room, seatIndex, err := manager.CreateRoom(CreateRoomInput{
		UserID:        "u1",
		PreferredSeat: &preferredSeat,
		BaseScore:     2,
	})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	if room.ID == "" {
		t.Fatal("room id should not be empty")
	}
	if room.Status != RoomStatusWaiting {
		t.Fatalf("room status = %q, want %q", room.Status, RoomStatusWaiting)
	}
	if room.Mode != DefaultMode {
		t.Fatalf("room mode = %q, want %q", room.Mode, DefaultMode)
	}
	if room.BaseScore != 2 {
		t.Fatalf("room base score = %d, want 2", room.BaseScore)
	}
	if seatIndex != preferredSeat {
		t.Fatalf("seat index = %d, want %d", seatIndex, preferredSeat)
	}
	if len(room.Seats) != 1 {
		t.Fatalf("seat count = %d, want 1", len(room.Seats))
	}
	if room.Seats[0].UserID != "u1" {
		t.Fatalf("seat user id = %q, want %q", room.Seats[0].UserID, "u1")
	}
}

func TestCreateRoomRejectsUserAlreadyInActiveRoom(t *testing.T) {
	manager := NewManager()
	if _, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"}); err != nil {
		t.Fatalf("first CreateRoom() error = %v", err)
	}

	_, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if !errors.Is(err, ErrUserAlreadyInActiveRoom) {
		t.Fatalf("second CreateRoom() error = %v, want ErrUserAlreadyInActiveRoom", err)
	}
}

func TestJoinRoomRejectsFullRoom(t *testing.T) {
	manager := NewManager()
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom u2 error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u3"}); err != nil {
		t.Fatalf("JoinRoom u3 error = %v", err)
	}

	_, _, err = manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u4"})
	if !errors.Is(err, ErrRoomFull) {
		t.Fatalf("JoinRoom() error = %v, want ErrRoomFull", err)
	}
}

func TestJoinRoomRejectsUserAlreadyInAnotherActiveRoom(t *testing.T) {
	manager := NewManager()
	roomA, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom roomA error = %v", err)
	}
	roomB, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u2"})
	if err != nil {
		t.Fatalf("CreateRoom roomB error = %v", err)
	}

	_, _, err = manager.JoinRoom(JoinRoomInput{RoomID: roomB.ID, UserID: "u1"})
	if !errors.Is(err, ErrUserAlreadyInActiveRoom) {
		t.Fatalf("JoinRoom() error = %v, want ErrUserAlreadyInActiveRoom", err)
	}

	snapshot, err := manager.GetRoom(roomA.ID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if len(snapshot.Seats) != 1 {
		t.Fatalf("roomA seat count = %d, want 1", len(snapshot.Seats))
	}
}

func TestQuickStartReturnsExistingWaitingRoom(t *testing.T) {
	manager := NewManager()
	room, seatIndex, err := manager.QuickStart(QuickStartInput{
		UserID:    "u1",
		BaseScore: 1,
	})
	if err != nil {
		t.Fatalf("first QuickStart() error = %v", err)
	}
	if seatIndex != 0 {
		t.Fatalf("first seat index = %d, want 0", seatIndex)
	}

	sameRoom, seatIndex, err := manager.QuickStart(QuickStartInput{
		UserID:    "u2",
		BaseScore: 1,
	})
	if err != nil {
		t.Fatalf("second QuickStart() error = %v", err)
	}
	if sameRoom.ID != room.ID {
		t.Fatalf("room id = %q, want %q", sameRoom.ID, room.ID)
	}
	if seatIndex != 1 {
		t.Fatalf("second seat index = %d, want 1", seatIndex)
	}
}

func TestQuickStartReturnsExistingSeatForSameUser(t *testing.T) {
	manager := NewManager()
	room, seatIndex, err := manager.QuickStart(QuickStartInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("first QuickStart() error = %v", err)
	}

	sameRoom, sameSeatIndex, err := manager.QuickStart(QuickStartInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("second QuickStart() error = %v", err)
	}
	if sameRoom.ID != room.ID {
		t.Fatalf("room id = %q, want %q", sameRoom.ID, room.ID)
	}
	if sameSeatIndex != seatIndex {
		t.Fatalf("seat index = %d, want %d", sameSeatIndex, seatIndex)
	}
}

func TestLeaveRoomRemovesUserFromWaitingRoom(t *testing.T) {
	manager := NewManager()
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	updatedRoom, err := manager.LeaveRoom(room.ID, "u2")
	if err != nil {
		t.Fatalf("LeaveRoom() error = %v", err)
	}
	if len(updatedRoom.Seats) != 1 {
		t.Fatalf("seat count = %d, want 1", len(updatedRoom.Seats))
	}
	if updatedRoom.Seats[0].UserID != "u1" {
		t.Fatalf("remaining user = %q, want %q", updatedRoom.Seats[0].UserID, "u1")
	}
}

func TestLeaveRoomClosesEmptyRoom(t *testing.T) {
	manager := NewManager()
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	leftRoom, err := manager.LeaveRoom(room.ID, "u1")
	if err != nil {
		t.Fatalf("LeaveRoom() error = %v", err)
	}
	if leftRoom.Status != RoomStatusClosed {
		t.Fatalf("room status = %q, want %q", leftRoom.Status, RoomStatusClosed)
	}

	_, err = manager.GetRoom(room.ID)
	if !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("GetRoom() error = %v, want ErrRoomNotFound", err)
	}
}

func TestLeaveRoomRejectsPlayingRoom(t *testing.T) {
	manager := NewManager()
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	manager.mu.Lock()
	manager.rooms[room.ID].Status = RoomStatusPlaying
	manager.mu.Unlock()

	_, err = manager.LeaveRoom(room.ID, "u1")
	if !errors.Is(err, ErrGameAlreadyStarted) {
		t.Fatalf("LeaveRoom() error = %v, want ErrGameAlreadyStarted", err)
	}
}

func TestJoinRoomHonorsPreferredSeat(t *testing.T) {
	manager := NewManager()
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	preferredSeat := 2

	updatedRoom, seatIndex, err := manager.JoinRoom(JoinRoomInput{
		RoomID:        room.ID,
		UserID:        "u2",
		PreferredSeat: &preferredSeat,
	})
	if err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}
	if seatIndex != 2 {
		t.Fatalf("seat index = %d, want 2", seatIndex)
	}
	if len(updatedRoom.Seats) != 2 {
		t.Fatalf("seat count = %d, want 2", len(updatedRoom.Seats))
	}
}

func TestJoinRoomRejectsUnavailablePreferredSeat(t *testing.T) {
	manager := NewManager()
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	preferredSeat := 0

	_, _, err = manager.JoinRoom(JoinRoomInput{
		RoomID:        room.ID,
		UserID:        "u2",
		PreferredSeat: &preferredSeat,
	})
	if !errors.Is(err, ErrSeatUnavailable) {
		t.Fatalf("JoinRoom() error = %v, want ErrSeatUnavailable", err)
	}
}
