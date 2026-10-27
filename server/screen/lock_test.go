package screen

import (
	"encoding/json"
	"testing"
	"time"
)

// statusOf drains a viewer's queue and returns the status documents it holds.
func statusOf(v *viewer) []Status {
	var out []Status
	for {
		select {
		case msg := <-v.send:
			if !msg.text {
				continue
			}
			var env struct {
				Type   string `json:"type"`
				Status Status `json:"status"`
			}
			if err := json.Unmarshal(msg.data, &env); err != nil || env.Type != "status" {
				continue
			}
			out = append(out, env.Status)
		default:
			return out
		}
	}
}

// A lock flip has to reach the page on its own: a locked host captures as a
// static dark image and refuses input, and a viewer that joined while the host
// was unlocked has no other way to learn why the mirror froze.
func TestLockFlipBroadcastsStatus(t *testing.T) {
	m, _, v := testManager(t)
	// No controller in this test: only the lock flip should emit a status.
	m.holder = nil
	m.holderSince = time.Time{}
	locked := false
	lockProbe = func() bool { return locked }
	t.Cleanup(func() { lockProbe = func() bool { return false } })

	m.housekeeping(m.gen)
	if got := statusOf(v); len(got) != 0 {
		t.Fatalf("unchanged lock state sent %d status messages, want 0", len(got))
	}

	locked = true
	m.housekeeping(m.gen)
	got := statusOf(v)
	if len(got) != 1 {
		t.Fatalf("locking sent %d status messages, want 1", len(got))
	}
	if !got[0].Input.Locked {
		t.Fatal("lock status does not mark the input as locked")
	}

	// The state is edge triggered: the page does not need a copy every second.
	m.housekeeping(m.gen)
	if again := statusOf(v); len(again) != 0 {
		t.Fatalf("steady locked state sent %d status messages, want 0", len(again))
	}

	locked = false
	m.housekeeping(m.gen)
	got = statusOf(v)
	if len(got) != 1 {
		t.Fatalf("unlocking sent %d status messages, want 1", len(got))
	}
	if got[0].Input.Locked {
		t.Fatal("unlock status still reports the input as locked")
	}
}

// Granting the Accessibility permission while the page is open must revive the
// "Take control" button without a reload: the page has no other trigger, and
// the server already knows because it probes the permission every second.
func TestPermissionGrantBroadcastsStatus(t *testing.T) {
	m, be, v := testManager(t)
	m.holder = nil
	m.holderSince = time.Time{}
	be.infoV = InputInfo{Available: false, Permission: "denied", CanPrompt: true, Backend: "cgevent"}
	m.inputOK = false

	m.housekeeping(m.gen)
	if got := statusOf(v); len(got) != 0 {
		t.Fatalf("unchanged permission sent %d status messages, want 0", len(got))
	}

	be.infoV = InputInfo{Available: true, Permission: "granted", Backend: "cgevent"}
	m.housekeeping(m.gen)
	got := statusOf(v)
	if len(got) != 1 {
		t.Fatalf("granting input permission sent %d status messages, want 1", len(got))
	}
	if !got[0].Input.Available {
		t.Fatal("status still reports input as unavailable after the grant")
	}

	m.housekeeping(m.gen)
	if again := statusOf(v); len(again) != 0 {
		t.Fatalf("steady permission state sent %d status messages, want 0", len(again))
	}
}
