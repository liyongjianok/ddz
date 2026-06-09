package record

import (
	"context"
	"errors"
	"testing"
	"time"

	"ddz/backend/internal/game"
	"ddz/backend/internal/room"
)

func TestServiceGetGameRecordRejectsNonParticipant(t *testing.T) {
	service := NewService(NewMemoryStore())
	err := service.store.SaveGameRecord(context.Background(), GameRecord{
		GameID:       "g1",
		RoomID:       "r1",
		Mode:         "classic",
		BaseScore:    1,
		Multiplier:   3,
		WinnerSide:   "landlord",
		StartedAt:    time.Now().UTC().Add(-time.Minute),
		EndedAt:      time.Now().UTC(),
		Participants: []string{"u1", "u2", "u3"},
		Players:      []RecordPlayer{{UserID: "u1", DisplayName: "u1", SeatIndex: 0, Role: "landlord", ScoreDelta: 6}},
	})
	if err != nil {
		t.Fatalf("SaveGameRecord() error = %v", err)
	}

	_, err = service.GetGameRecord(context.Background(), "u9", "g1")
	if !errors.Is(err, ErrRecordForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrRecordForbidden)
	}
}

func TestServiceSaveGameRecordBuildsSettlementRecord(t *testing.T) {
	service := NewService(NewMemoryStore())
	currentRoom := &room.Room{
		ID:        "r1",
		Mode:      "classic",
		BaseScore: 2,
		CurrentGame: &game.Game{
			ID:        "g1",
			StartedAt: time.Now().UTC().Add(-time.Minute),
			EndedAt:   time.Now().UTC(),
			Players: []game.PlayerState{
				{UserID: "u1", SeatIndex: 0, Role: game.RoleLandlord},
				{UserID: "u2", SeatIndex: 1, Role: game.RoleFarmer},
				{UserID: "u3", SeatIndex: 2, Role: game.RoleFarmer},
			},
			Settlement: &game.SettlementResult{
				BaseScore:  2,
				Multiplier: 3,
				WinnerSide: game.WinnerSideLandlord,
				Players: []game.SettlementPlayer{
					{UserID: "u1", SeatIndex: 0, Role: game.RoleLandlord, DeltaScore: 12, IsWinner: true},
					{UserID: "u2", SeatIndex: 1, Role: game.RoleFarmer, DeltaScore: -6},
					{UserID: "u3", SeatIndex: 2, Role: game.RoleFarmer, DeltaScore: -6},
				},
			},
		},
	}

	if err := service.SaveGameRecord(context.Background(), currentRoom); err != nil {
		t.Fatalf("SaveGameRecord() error = %v", err)
	}

	record, err := service.GetGameRecord(context.Background(), "u1", "g1")
	if err != nil {
		t.Fatalf("GetGameRecord() error = %v", err)
	}
	if record.WinnerSide != "landlord" {
		t.Fatalf("winner_side = %q, want %q", record.WinnerSide, "landlord")
	}
	if len(record.Players) != 3 {
		t.Fatalf("players len = %d, want 3", len(record.Players))
	}
}
