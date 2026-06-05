package room

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetLobbySummaryAggregatesRoomsByModeAndBaseScore(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})

	roomA, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1", BaseScore: 1, Mode: "classic"})
	if err != nil {
		t.Fatalf("CreateRoom roomA error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: roomA.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom roomA error = %v", err)
	}

	if _, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u3", BaseScore: 1, Mode: "classic"}); err != nil {
		t.Fatalf("CreateRoom roomB error = %v", err)
	}

	roomC, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u4", BaseScore: 2, Mode: "classic"})
	if err != nil {
		t.Fatalf("CreateRoom roomC error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: roomC.ID, UserID: "u5"}); err != nil {
		t.Fatalf("JoinRoom roomC u5 error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: roomC.ID, UserID: "u6"}); err != nil {
		t.Fatalf("JoinRoom roomC u6 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: roomC.ID, UserID: "u4", Ready: true}); err != nil {
		t.Fatalf("Ready roomC u4 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: roomC.ID, UserID: "u5", Ready: true}); err != nil {
		t.Fatalf("Ready roomC u5 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: roomC.ID, UserID: "u6", Ready: true}); err != nil {
		t.Fatalf("Ready roomC u6 error = %v", err)
	}

	summary, err := manager.GetLobbySummary()
	if err != nil {
		t.Fatalf("GetLobbySummary() error = %v", err)
	}

	if summary.OnlinePlayers != 6 {
		t.Fatalf("online players = %d, want 6", summary.OnlinePlayers)
	}
	if summary.ActiveRooms != 3 {
		t.Fatalf("active rooms = %d, want 3", summary.ActiveRooms)
	}
	if len(summary.Modes) != 2 {
		t.Fatalf("mode summary len = %d, want 2", len(summary.Modes))
	}

	if summary.Modes[0].Mode != "classic" || summary.Modes[0].BaseScore != 1 {
		t.Fatalf("first mode summary = %+v, want classic/1", summary.Modes[0])
	}
	if summary.Modes[0].OnlinePlayers != 3 {
		t.Fatalf("first mode online players = %d, want 3", summary.Modes[0].OnlinePlayers)
	}
	if summary.Modes[0].WaitingRooms != 2 {
		t.Fatalf("first mode waiting rooms = %d, want 2", summary.Modes[0].WaitingRooms)
	}

	if summary.Modes[1].Mode != "classic" || summary.Modes[1].BaseScore != 2 {
		t.Fatalf("second mode summary = %+v, want classic/2", summary.Modes[1])
	}
	if summary.Modes[1].OnlinePlayers != 3 {
		t.Fatalf("second mode online players = %d, want 3", summary.Modes[1].OnlinePlayers)
	}
	if summary.Modes[1].WaitingRooms != 0 {
		t.Fatalf("second mode waiting rooms = %d, want 0", summary.Modes[1].WaitingRooms)
	}
}

func TestListRoomsFiltersAndPaginatesWithoutHiddenCards(t *testing.T) {
	manager := NewManagerWithRNG(&fixedRNG{value: 0})

	roomA, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u1", BaseScore: 1, Mode: "classic"})
	if err != nil {
		t.Fatalf("CreateRoom roomA error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: roomA.ID, UserID: "u2"}); err != nil {
		t.Fatalf("JoinRoom roomA error = %v", err)
	}

	roomB, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u3", BaseScore: 1, Mode: "classic"})
	if err != nil {
		t.Fatalf("CreateRoom roomB error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: roomB.ID, UserID: "u4"}); err != nil {
		t.Fatalf("JoinRoom roomB u4 error = %v", err)
	}
	if _, _, err := manager.JoinRoom(JoinRoomInput{RoomID: roomB.ID, UserID: "u5"}); err != nil {
		t.Fatalf("JoinRoom roomB u5 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: roomB.ID, UserID: "u3", Ready: true}); err != nil {
		t.Fatalf("Ready roomB u3 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: roomB.ID, UserID: "u4", Ready: true}); err != nil {
		t.Fatalf("Ready roomB u4 error = %v", err)
	}
	if _, _, _, err := manager.Ready(ReadyInput{RoomID: roomB.ID, UserID: "u5", Ready: true}); err != nil {
		t.Fatalf("Ready roomB u5 error = %v", err)
	}

	if _, _, err := manager.CreateRoom(CreateRoomInput{UserID: "u6", BaseScore: 3, Mode: "ranked"}); err != nil {
		t.Fatalf("CreateRoom roomC error = %v", err)
	}

	result, err := manager.ListRooms(RoomListFilter{
		Mode:     "classic",
		Status:   RoomStatusWaiting,
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("ListRooms() error = %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item len = %d, want 1", len(result.Items))
	}
	if result.Items[0].RoomID != roomA.ID {
		t.Fatalf("room id = %q, want %q", result.Items[0].RoomID, roomA.ID)
	}
	if result.Items[0].PlayerCount != 2 {
		t.Fatalf("player count = %d, want 2", result.Items[0].PlayerCount)
	}
	if result.Items[0].Status != string(RoomStatusWaiting) {
		t.Fatalf("status = %q, want %q", result.Items[0].Status, RoomStatusWaiting)
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(payload) == "" {
		t.Fatal("payload should not be empty")
	}
	if containsHiddenCards(payload) {
		t.Fatal("room list should not contain hidden card fields")
	}

	paged, err := manager.ListRooms(RoomListFilter{
		Page:     2,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListRooms paged error = %v", err)
	}
	if paged.Total != 3 {
		t.Fatalf("paged total = %d, want 3", paged.Total)
	}
	if len(paged.Items) != 1 {
		t.Fatalf("paged item len = %d, want 1", len(paged.Items))
	}
	if paged.Items[0].RoomID != roomB.ID {
		t.Fatalf("second page room id = %q, want %q", paged.Items[0].RoomID, roomB.ID)
	}
}

func containsHiddenCards(payload []byte) bool {
	text := string(payload)
	return containsAny(text, `"current_game"`, `"hand"`, `"bottom_cards"`, `"last_play"`, `"deadline_at"`)
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if len(value) > 0 && jsonContains(text, value) {
			return true
		}
	}
	return false
}

func jsonContains(text string, value string) bool {
	return strings.Contains(text, value)
}
