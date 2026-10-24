package portforward

import "testing"

func TestAdoptKeepPIDKeepsAliveRecorded(t *testing.T) {
	alive := func(pid int) bool { return pid == 22667 }
	got := adoptKeepPID(22667, []int{21818, 22667}, alive)
	if got != 22667 {
		t.Fatalf("got %d, want recorded live pid", got)
	}
}

func TestAdoptKeepPIDAdoptsNewestWhenRecordedDead(t *testing.T) {
	alive := func(int) bool { return false }
	got := adoptKeepPID(15142, []int{21818, 22667}, alive)
	if got != 22667 {
		t.Fatalf("got %d, want newest live pid 22667", got)
	}
}

func TestAdoptKeepPIDEmptyLive(t *testing.T) {
	got := adoptKeepPID(15142, nil, func(int) bool { return false })
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}
