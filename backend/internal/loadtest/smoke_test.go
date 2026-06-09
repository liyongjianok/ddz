package loadtest

import "testing"

func TestBuildRoomPlans(t *testing.T) {
	plans := buildRoomPlans(8)
	if len(plans) != 3 {
		t.Fatalf("plan count = %d, want 3", len(plans))
	}

	want := []int{3, 3, 2}
	for idx, plan := range plans {
		if plan.Index != idx {
			t.Fatalf("plan[%d].Index = %d, want %d", idx, plan.Index, idx)
		}
		if plan.PlayerCount != want[idx] {
			t.Fatalf("plan[%d].PlayerCount = %d, want %d", idx, plan.PlayerCount, want[idx])
		}
	}
}

func TestBuildWSURL(t *testing.T) {
	wsURL, err := buildWSURL("http://127.0.0.1:8080", "r_000123", "token_abc")
	if err != nil {
		t.Fatalf("buildWSURL() error = %v", err)
	}

	want := "ws://127.0.0.1:8080/ws/v1/rooms/r_000123?token=token_abc"
	if wsURL != want {
		t.Fatalf("wsURL = %q, want %q", wsURL, want)
	}
}
