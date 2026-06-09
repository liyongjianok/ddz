package app

import (
	"context"
	"testing"
	"time"

	"ddz/backend/internal/storage/redisstore"
)

func TestNewRedisStoreReturnsUsableStore(t *testing.T) {
	store := NewRedisStore(testConfig())

	connection := redisstore.PlayerConnection{
		UserID:       "u1",
		RoomID:       "r1",
		ConnectionID: "c1",
		UpdatedAt:    time.Now().UTC(),
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
}
