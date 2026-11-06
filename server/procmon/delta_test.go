package procmon

import (
	"testing"
	"time"
)

func TestBuildSnapshotCPUDelta(t *testing.T) {
	prev := []rawProc{
		{PID: 8375, PPID: 1, User: "xhd2015", RSSKiB: 1000, CPUTime: 10 * time.Second, Cmd: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
		{PID: 99, PPID: 1, User: "xhd2015", RSSKiB: 200, CPUTime: 1 * time.Second, Cmd: "ai-critic"},
	}
	cur := []rawProc{
		{PID: 8375, PPID: 1, User: "xhd2015", RSSKiB: 1600000, CPUTime: 12 * time.Second, Cmd: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome --type=renderer"},
		{PID: 99, PPID: 1, User: "xhd2015", RSSKiB: 200, CPUTime: 1 * time.Second, Cmd: "ai-critic"},
	}
	at := time.Date(2026, 11, 6, 13, 0, 0, 0, time.FixedZone("CST", 8*3600))
	snap := buildSnapshot(prev, cur, time.Second, at, Memory{Total: 16 << 30, Used: 15 << 30}, Load{M1: 22.8}, 99, "xhd2015", 10)
	if snap.IntervalMs != 1000 {
		t.Fatalf("interval=%d", snap.IntervalMs)
	}
	if snap.SelfPID != 99 {
		t.Fatalf("self=%d", snap.SelfPID)
	}
	if len(snap.Processes) != 2 {
		t.Fatalf("len=%d", len(snap.Processes))
	}
	chrome := snap.Processes[0]
	if chrome.PID != 8375 {
		t.Fatalf("sorted first = %+v", chrome)
	}
	// 2.0s CPU in 1.0s wall = 200% of one core
	if chrome.CPUPercent < 199.9 || chrome.CPUPercent > 200.1 {
		t.Fatalf("cpu=%v", chrome.CPUPercent)
	}
	if chrome.Name != "Google Chrome" {
		t.Fatalf("name=%q", chrome.Name)
	}
	if chrome.Protected != "" {
		t.Fatalf("chrome protected %q", chrome.Protected)
	}
	self := snap.Processes[1]
	if self.Protected == "" {
		t.Fatalf("self should be protected")
	}
	if snap.CPU.UsedPercent < 19.9 || snap.CPU.UsedPercent > 20.1 {
		t.Fatalf("host cpu=%v (200%% of 1 core / 10 cores)", snap.CPU.UsedPercent)
	}
}

func TestSamplerSingleflightAndCache(t *testing.T) {
	now := time.Date(2026, 11, 6, 13, 0, 0, 0, time.UTC)
	n := 0
	sampleA := "  PID PPID UID USER STAT RSS ELAPSED TIME COMMAND\n  10 1 501 me S 100 00:01 0:01.00 /bin/sleep 1\n"
	sampleB := "  PID PPID UID USER STAT RSS ELAPSED TIME COMMAND\n  10 1 501 me S 100 00:01 0:01.20 /bin/sleep 1\n"
	s := &sampler{env: Env{
		Now: func() time.Time { return now },
		Sleep: func(d time.Duration) {
			now = now.Add(d)
		},
		PS: func() (string, error) {
			n++
			if n <= 1 {
				return sampleA, nil
			}
			return sampleB, nil
		},
		Load:     func() (Load, error) { return Load{M1: 1}, nil },
		Memory:   func() Memory { return Memory{Total: 1} },
		NumCPU:   2,
		SelfPID:  1,
		Username: "me",
	}}
	// fix typo - I accidentally wrote " mar"
	_ = n
	first, err := s.get()
	if err != nil {
		t.Fatal(err)
	}
	if first.Processes[0].CPUPercent < 99 || first.Processes[0].CPUPercent > 101 {
		t.Fatalf("cpu=%v interval=%d psCalls=%d", first.Processes[0].CPUPercent, first.IntervalMs, n)
	}
	calls := n
	second, err := s.get()
	if err != nil {
		t.Fatal(err)
	}
	if n != calls {
		t.Fatalf("cache missed: ps calls %d -> %d", calls, n)
	}
	if second != first {
		t.Fatalf("expected same cached pointer")
	}
}
