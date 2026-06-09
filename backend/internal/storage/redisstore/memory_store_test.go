package redisstore

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStorePlayerConnectionLifecycle(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	connection := PlayerConnection{
		UserID:       "u1",
		RoomID:       "r1",
		ConnectionID: "c1",
		UpdatedAt:    now,
	}

	if err := store.SavePlayerConnection(context.Background(), connection, time.Minute); err != nil {
		t.Fatalf("SavePlayerConnection() error = %v", err)
	}

	got, ok, err := store.GetPlayerConnection(context.Background(), "u1", "r1")
	if err != nil {
		t.Fatalf("GetPlayerConnection() error = %v", err)
	}
	if !ok {
		t.Fatal("GetPlayerConnection() ok = false, want true")
	}
	if got.ConnectionID != "c1" {
		t.Fatalf("connection id = %q, want %q", got.ConnectionID, "c1")
	}

	if err := store.DeletePlayerConnection(context.Background(), "u1", "r1"); err != nil {
		t.Fatalf("DeletePlayerConnection() error = %v", err)
	}
	_, ok, err = store.GetPlayerConnection(context.Background(), "u1", "r1")
	if err != nil {
		t.Fatalf("GetPlayerConnection() after delete error = %v", err)
	}
	if ok {
		t.Fatal("GetPlayerConnection() ok = true after delete, want false")
	}
}

func TestMemoryStoreReconnectStateExpiresByTTL(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	state := ReconnectState{
		UserID:         "u1",
		RoomID:         "r1",
		SeatIndex:      1,
		GameID:         "g1",
		DisconnectedAt: now,
		ExpiresAt:      now.Add(time.Minute),
	}

	if err := store.SaveReconnectState(context.Background(), state, time.Minute); err != nil {
		t.Fatalf("SaveReconnectState() error = %v", err)
	}

	got, ok, err := store.GetReconnectState(context.Background(), "u1", "r1")
	if err != nil {
		t.Fatalf("GetReconnectState() error = %v", err)
	}
	if !ok {
		t.Fatal("GetReconnectState() ok = false, want true")
	}
	if got.GameID != "g1" {
		t.Fatalf("game id = %q, want %q", got.GameID, "g1")
	}

	store.now = func() time.Time { return now.Add(2 * time.Minute) }
	_, ok, err = store.GetReconnectState(context.Background(), "u1", "r1")
	if err != nil {
		t.Fatalf("GetReconnectState() expired error = %v", err)
	}
	if ok {
		t.Fatal("GetReconnectState() ok = true after ttl, want false")
	}
}

func TestMemoryStoreRoomSnapshotLifecycle(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	snapshot := RoomSnapshotState{
		RoomID:    "r1",
		Payload:   []byte(`{"room_id":"r1"}`),
		UpdatedAt: now,
		ExpiresAt: now.Add(5 * time.Minute),
	}

	if err := store.SaveRoomSnapshot(context.Background(), snapshot, 5*time.Minute); err != nil {
		t.Fatalf("SaveRoomSnapshot() error = %v", err)
	}

	got, ok, err := store.GetRoomSnapshot(context.Background(), "r1")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() error = %v", err)
	}
	if !ok {
		t.Fatal("GetRoomSnapshot() ok = false, want true")
	}
	if string(got.Payload) != `{"room_id":"r1"}` {
		t.Fatalf("payload = %q, want %q", string(got.Payload), `{"room_id":"r1"}`)
	}

	if err := store.DeleteRoomSnapshot(context.Background(), "r1"); err != nil {
		t.Fatalf("DeleteRoomSnapshot() error = %v", err)
	}
	_, ok, err = store.GetRoomSnapshot(context.Background(), "r1")
	if err != nil {
		t.Fatalf("GetRoomSnapshot() after delete error = %v", err)
	}
	if ok {
		t.Fatal("GetRoomSnapshot() ok = true after delete, want false")
	}
}
