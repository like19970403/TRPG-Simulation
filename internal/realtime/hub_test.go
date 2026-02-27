package realtime

import (
	"testing"
	"time"
)

func TestHub_GetOrCreateRoom_CreatesNewRoom(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	room := hub.GetOrCreateRoom("sess-1", "gm-1")
	if room == nil {
		t.Fatal("expected room, got nil")
	}
	if room.SessionID() != "sess-1" {
		t.Errorf("SessionID = %q, want %q", room.SessionID(), "sess-1")
	}
	if hub.RoomCount() != 1 {
		t.Errorf("RoomCount = %d, want 1", hub.RoomCount())
	}
}

func TestHub_GetOrCreateRoom_ReturnsExisting(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	room1 := hub.GetOrCreateRoom("sess-1", "gm-1")
	room2 := hub.GetOrCreateRoom("sess-1", "gm-1")

	if room1 != room2 {
		t.Error("expected same room instance")
	}
	if hub.RoomCount() != 1 {
		t.Errorf("RoomCount = %d, want 1", hub.RoomCount())
	}
}

func TestHub_GetRoom_ReturnsNilIfNotExists(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	room := hub.GetRoom("nonexistent")
	if room != nil {
		t.Error("expected nil, got room")
	}
}

func TestHub_RemoveRoom(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	hub.GetOrCreateRoom("sess-1", "gm-1")
	if hub.RoomCount() != 1 {
		t.Fatalf("RoomCount = %d, want 1", hub.RoomCount())
	}

	hub.RemoveRoom("sess-1")
	time.Sleep(50 * time.Millisecond)

	if hub.RoomCount() != 0 {
		t.Errorf("RoomCount = %d, want 0", hub.RoomCount())
	}
	if hub.GetRoom("sess-1") != nil {
		t.Error("room should be nil after removal")
	}
}

func TestHub_Stop_CleansUpAllRooms(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())

	hub.GetOrCreateRoom("sess-1", "gm-1")
	hub.GetOrCreateRoom("sess-2", "gm-2")

	if hub.RoomCount() != 2 {
		t.Fatalf("RoomCount = %d, want 2", hub.RoomCount())
	}

	hub.Stop()

	if hub.RoomCount() != 0 {
		t.Errorf("RoomCount = %d, want 0 after Stop", hub.RoomCount())
	}
}
