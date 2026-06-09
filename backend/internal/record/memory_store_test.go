package record

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreAppendAndListRecords(t *testing.T) {
	store := NewMemoryStore()

	event := Event{
		GameID:    "g1",
		RoomID:    "r1",
		Seq:       1,
		EventType: "bid_placed",
		Payload: map[string]any{
			"score": 3,
		},
		CreatedAt: time.Now().UTC(),
	}
	if err := store.AppendEvent(context.Background(), event); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	record := GameRecord{
		GameID:       "g1",
		RoomID:       "r1",
		Mode:         "classic",
		BaseScore:    1,
		Multiplier:   3,
		WinnerSide:   "landlord",
		StartedAt:    time.Now().UTC().Add(-time.Minute),
		EndedAt:      time.Now().UTC(),
		Participants: []string{"u1", "u2", "u3"},
		Players: []RecordPlayer{
			{UserID: "u1", DisplayName: "u1", SeatIndex: 0, Role: "landlord", ScoreDelta: 6},
			{UserID: "u2", DisplayName: "u2", SeatIndex: 1, Role: "farmer", ScoreDelta: -3},
			{UserID: "u3", DisplayName: "u3", SeatIndex: 2, Role: "farmer", ScoreDelta: -3},
		},
	}
	if err := store.SaveGameRecord(context.Background(), record); err != nil {
		t.Fatalf("SaveGameRecord() error = %v", err)
	}

	got, ok, err := store.GetGameRecord(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetGameRecord() error = %v", err)
	}
	if !ok {
		t.Fatal("GetGameRecord() ok = false, want true")
	}
	if len(got.Events) != 1 {
		t.Fatalf("events len = %d, want 1", len(got.Events))
	}

	list, err := store.ListRecordsByUser(context.Background(), "u1", 1, 20)
	if err != nil {
		t.Fatalf("ListRecordsByUser() error = %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("total = %d, want 1", list.Total)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(list.Items))
	}
	if list.Items[0].ScoreDelta != 6 {
		t.Fatalf("score_delta = %d, want 6", list.Items[0].ScoreDelta)
	}
}
