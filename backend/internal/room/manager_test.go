package room

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"ddz/backend/internal/game"
)

func TestCreateRoomAssignsCreatorSeat(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	if _, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"}); err != nil {
		t.Fatalf("first CreateRoom() error = %v", err)
	}

	_, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if !errors.Is(err, ErrUserAlreadyInActiveRoom) {
		t.Fatalf("second CreateRoom() error = %v, want ErrUserAlreadyInActiveRoom", err)
	}
}

func TestJoinRoomRejectsFullRoom(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u3"}); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready u1 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u2", Ready: true}); err != nil {
		t.Fatalf("Ready u2 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u3", Ready: true}); err != nil {
		t.Fatalf("Ready u3 error = %v", err)
	}

	_, err = manager.LeaveRoom(room.ID, "u1")
	if !errors.Is(err, ErrGameAlreadyStarted) {
		t.Fatalf("LeaveRoom() error = %v, want ErrGameAlreadyStarted", err)
	}
}

func TestJoinRoomHonorsPreferredSeat(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
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

func TestReadyStartsGameWhenRoomIsFullAndAllReady(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 1})
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

	if updatedRoom, _, started, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready u1 error = %v", err)
	} else if started {
		t.Fatal("room should not start before all players are ready")
	} else if !findSeatReady(updatedRoom.Seats, "u1") {
		t.Fatal("u1 should be ready")
	}
	if _, _, started, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u2", Ready: true}); err != nil {
		t.Fatalf("Ready u2 error = %v", err)
	} else if started {
		t.Fatal("room should not start before all players are ready")
	}

	updatedRoom, seatIndex, started, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u3", Ready: true})
	if err != nil {
		t.Fatalf("Ready u3 error = %v", err)
	}
	if !started {
		t.Fatal("room should start after all players are ready")
	}
	if seatIndex != 2 {
		t.Fatalf("seat index = %d, want 2", seatIndex)
	}
	if updatedRoom.Status != RoomStatusPlaying {
		t.Fatalf("room status = %q, want %q", updatedRoom.Status, RoomStatusPlaying)
	}
	if updatedRoom.CurrentGame == nil {
		t.Fatal("current game should be created")
	}
	if updatedRoom.CurrentGame.Phase != game.GamePhaseBidding {
		t.Fatalf("game phase = %q, want %q", updatedRoom.CurrentGame.Phase, game.GamePhaseBidding)
	}
	if len(updatedRoom.CurrentGame.Players) != game.PlayerCount {
		t.Fatalf("player count = %d, want %d", len(updatedRoom.CurrentGame.Players), game.PlayerCount)
	}
}

func TestFillRobotsStartsGameForSingleReadyPlayer(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}

	updatedRoom, started, err := manager.FillRobots(room.ID)
	if err != nil {
		t.Fatalf("FillRobots() error = %v", err)
	}
	if !started {
		t.Fatal("room should start after robots fill empty seats")
	}
	if updatedRoom.Status != RoomStatusPlaying {
		t.Fatalf("room status = %q, want %q", updatedRoom.Status, RoomStatusPlaying)
	}
	if len(updatedRoom.Seats) != game.PlayerCount {
		t.Fatalf("seat count = %d, want %d", len(updatedRoom.Seats), game.PlayerCount)
	}

	robotCount := 0
	for _, seat := range updatedRoom.Seats {
		if !seat.Ready {
			t.Fatalf("seat %d should be ready", seat.SeatIndex)
		}
		if seat.IsRobot {
			robotCount++
			if !strings.HasPrefix(seat.UserID, "robot_") {
				t.Fatalf("robot user id = %q, want robot_ prefix", seat.UserID)
			}
		}
	}
	if robotCount != 2 {
		t.Fatalf("robot count = %d, want 2", robotCount)
	}
	if updatedRoom.CurrentGame == nil {
		t.Fatal("current game should be created")
	}
	for _, player := range updatedRoom.CurrentGame.Players {
		if strings.HasPrefix(player.UserID, "robot_") && !player.IsRobot {
			t.Fatalf("game player %q should be marked as robot", player.UserID)
		}
	}
}

func TestFillRobotsIsNoopAfterRoomStarted(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
	if _, started, err := manager.FillRobots(room.ID); err != nil {
		t.Fatalf("FillRobots() error = %v", err)
	} else if !started {
		t.Fatal("room should start after first FillRobots()")
	}

	updatedRoom, started, err := manager.FillRobots(room.ID)
	if err != nil {
		t.Fatalf("second FillRobots() error = %v", err)
	}
	if started {
		t.Fatal("second FillRobots() should not report a new start")
	}
	if len(updatedRoom.Seats) != game.PlayerCount {
		t.Fatalf("seat count = %d, want %d", len(updatedRoom.Seats), game.PlayerCount)
	}
}

func TestHandleRobotTurnUsesLegalAutoAction(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 1})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
	startedRoom, started, err := manager.FillRobots(room.ID)
	if err != nil {
		t.Fatalf("FillRobots() error = %v", err)
	}
	if !started {
		t.Fatal("room should start")
	}
	currentSeat := startedRoom.CurrentGame.CurrentSeatIndex
	if !startedRoom.CurrentGame.Players[currentSeat].IsRobot {
		t.Fatalf("current seat %d should be robot for this test", currentSeat)
	}

	updatedRoom, action, err := manager.HandleRobotTurn(room.ID)
	if err != nil {
		t.Fatalf("HandleRobotTurn() error = %v", err)
	}
	if action != TimeoutActionAutoBid {
		t.Fatalf("action = %q, want %q", action, TimeoutActionAutoBid)
	}
	if len(updatedRoom.CurrentGame.BiddingState.Bids) != 1 {
		t.Fatalf("bid count = %d, want 1", len(updatedRoom.CurrentGame.BiddingState.Bids))
	}
	if updatedRoom.CurrentGame.BiddingState.Bids[0].SeatIndex != currentSeat {
		t.Fatalf("bid seat = %d, want %d", updatedRoom.CurrentGame.BiddingState.Bids[0].SeatIndex, currentSeat)
	}
}

func TestHandleRobotTurnNoopsForHumanTurn(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true}); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
	if _, started, err := manager.FillRobots(room.ID); err != nil {
		t.Fatalf("FillRobots() error = %v", err)
	} else if !started {
		t.Fatal("room should start")
	}

	updatedRoom, action, err := manager.HandleRobotTurn(room.ID)
	if err != nil {
		t.Fatalf("HandleRobotTurn() error = %v", err)
	}
	if action != TimeoutActionNone {
		t.Fatalf("action = %q, want %q", action, TimeoutActionNone)
	}
	if len(updatedRoom.CurrentGame.BiddingState.Bids) != 0 {
		t.Fatalf("bid count = %d, want 0", len(updatedRoom.CurrentGame.BiddingState.Bids))
	}
}

func TestReadyRejectsUserNotInRoom(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	_, _, _, err = manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u2", Ready: true})
	if !errors.Is(err, ErrUserNotInRoom) {
		t.Fatalf("Ready() error = %v, want ErrUserNotInRoom", err)
	}
}

func TestConcurrentJoinAndReadyKeepsRoomConsistent(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})
	room, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u2"})
		errCh <- err
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, err := manager.JoinRoom(JoinRoomInput{RoomID: room.ID, UserID: "u3"})
		errCh <- err
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true})
		errCh <- err
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, _, err := manager.Ready(ReadyInput{RoomID: room.ID, UserID: "u1", Ready: true})
		errCh <- err
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil && !errors.Is(err, ErrUserNotInRoom) {
			t.Fatalf("unexpected error = %v", err)
		}
	}

	snapshot, err := manager.GetRoom(room.ID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if len(snapshot.Seats) < 1 || len(snapshot.Seats) > game.PlayerCount {
		t.Fatalf("seat count = %d, want within [1,%d]", len(snapshot.Seats), game.PlayerCount)
	}

	seenUsers := make(map[string]struct{}, len(snapshot.Seats))
	seenSeats := make(map[int]struct{}, len(snapshot.Seats))
	for _, seat := range snapshot.Seats {
		if _, exists := seenUsers[seat.UserID]; exists {
			t.Fatalf("duplicate user in room: %s", seat.UserID)
		}
		if _, exists := seenSeats[seat.SeatIndex]; exists {
			t.Fatalf("duplicate seat in room: %d", seat.SeatIndex)
		}
		seenUsers[seat.UserID] = struct{}{}
		seenSeats[seat.SeatIndex] = struct{}{}
	}
}

func findSeatReady(seats []Seat, userID string) bool {
	for _, seat := range seats {
		if seat.UserID == userID {
			return seat.Ready
		}
	}
	return false
}

type fixedRNG struct {
	value int
}

func (r *fixedRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	value := r.value % n
	if value < 0 {
		value += n
	}
	return value
}
