package procmon

import (
	"strings"
	"testing"
	"time"
)

func TestParsePSClock(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"0:00.12", 120 * time.Millisecond},
		{"1:23.45", time.Minute + 23450*time.Millisecond},
		{"214:21.53", 214*time.Minute + 21530*time.Millisecond},
		{"1:02:03", time.Hour + 2*time.Minute + 3*time.Second},
		{"07-22:49:36", 7*24*time.Hour + 22*time.Hour + 49*time.Minute + 36*time.Second},
		{"00:00", 0},
	}
	for _, c := range cases {
		got, err := parsePSClock(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("%s: got %v want %v", c.in, got, c.want)
		}
	}
}

func TestParsePSTableKeepsSpacedCommand(t *testing.T) {
	out := strings.Join([]string{
		"  PID  PPID   UID USER             STAT    RSS     ELAPSED      TIME COMMAND",
		"    1     0     0 root             Ss    14000 07-22:49:36 214:21.38 /sbin/launchd",
		"  591     1   201 _windowserver    Ss    80000 07-22:49:16  12:04.00 /System/Library/PrivateFrameworks/SkyLight.framework/Resources/WindowServer -daemon",
		" 8375     1   501 xhd2015          R    1600000    13:12:00   1:23.00 /Applications/Google Chrome.app/Contents/MacOS/Google Chrome --type=renderer",
		"27937     1   501 xhd2015          S    340000     04:00:00   0:40.00 eluc-wt-bin --port 8466",
	}, "\n")
	procs, err := parsePSTable(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(procs) != 4 {
		t.Fatalf("len=%d", len(procs))
	}
	chrome := procs[2]
	if chrome.PID != 8375 || chrome.User != "xhd2015" {
		t.Fatalf("chrome = %+v", chrome)
	}
	if displayName(chrome.Cmd) != "Google Chrome" {
		t.Fatalf("name=%q cmd=%q", displayName(chrome.Cmd), chrome.Cmd)
	}
	if !strings.Contains(chrome.Cmd, "--type=renderer") {
		t.Fatalf("cmd lost flags: %q", chrome.Cmd)
	}
	if displayName(procs[1].Cmd) != "WindowServer" {
		t.Fatalf("windowserver name=%q", displayName(procs[1].Cmd))
	}
}

func TestDisplayNameSpacedPath(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
	}{
		{"/Library/Application Support/Adobe/Adobe Desktop Common/ElevationManager/AdobeUpdateService", "AdobeUpdateService"},
		{"/Library/Application Support/Adobe/Adobe Desktop Common/IPCBox/AdobeIPCBroker.app/Contents/MacOS/AdobeIPCBroker --launchedBy=launchd", "AdobeIPCBroker"},
		{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome --type=renderer", "Google Chrome"},
		{"eluc-wt-bin --port 8466", "eluc-wt-bin"},
		{"/sbin/launchd", "launchd"},
	}
	for _, c := range cases {
		got := displayName(c.cmd)
		if got != c.want {
			t.Fatalf("displayName(%q)=%q want %q", c.cmd, got, c.want)
		}
	}
}

func TestParseLoadAvgDarwin(t *testing.T) {
	load, err := parseLoadAvg("{ 22.80 24.70 19.70 }")
	if err != nil {
		t.Fatal(err)
	}
	if load.M1 != 22.8 || load.M5 != 24.7 || load.M15 != 19.7 {
		t.Fatalf("%+v", load)
	}
}

func TestParseSwapUsage(t *testing.T) {
	total, used, free := parseSwapUsage("total = 8192.00M  used = 6526.31M  free = 1665.69M  (encrypted)")
	if total != 8192<<20 {
		t.Fatalf("total=%d", total)
	}
	if used < 6500<<20 || used > 6600<<20 {
		t.Fatalf("used=%d", used)
	}
	if free < 1600<<20 || free > 1700<<20 {
		t.Fatalf("free=%d", free)
	}
}

func TestParseLinuxMeminfo(t *testing.T) {
	mem := parseLinuxMeminfo("MemTotal:        8000000 kB\nMemFree:          500000 kB\nMemAvailable:    2000000 kB\nSwapTotal:       1000000 kB\nSwapFree:         250000 kB\n")
	if mem.Total != 8000000*1024 {
		t.Fatalf("total=%d", mem.Total)
	}
	if mem.Used != 6000000*1024 {
		t.Fatalf("used=%d", mem.Used)
	}
	if mem.SwapUsed != 750000*1024 {
		t.Fatalf("swap used=%d", mem.SwapUsed)
	}
}

func TestParseRealPS(t *testing.T) {
	out, err := runPS()
	if err != nil {
		t.Fatal(err)
	}
	procs, err := parsePSTable(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(procs) < 2 {
		t.Fatalf("len=%d", len(procs))
	}
	seenSelf := false
	self := defaultEnv().SelfPID
	for _, p := range procs {
		if p.PID == self {
			seenSelf = true
		}
		if p.Cmd == "" {
			t.Fatalf("empty cmd pid=%d", p.PID)
		}
	}
	if !seenSelf {
		t.Fatalf("did not find self pid %d", self)
	}
}
