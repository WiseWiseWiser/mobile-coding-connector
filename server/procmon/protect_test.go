package procmon

import "testing"

func TestProtectReason(t *testing.T) {
	self := map[int]struct{}{99: {}, 50: {}}
	cases := []struct {
		p    rawProc
		want string
	}{
		{rawProc{PID: 1, Cmd: "/sbin/launchd"}, "init (PID <= 1)"},
		{rawProc{PID: 0, Cmd: "kernel_task"}, "init (PID <= 1)"},
		{rawProc{PID: 99, Cmd: "ai-critic"}, "this server (self or ancestor)"},
		{rawProc{PID: 591, Cmd: "/System/Library/PrivateFrameworks/SkyLight.framework/Resources/WindowServer -daemon"}, "system-critical (WindowServer)"},
		{rawProc{PID: 8375, Cmd: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}, ""},
	}
	for _, c := range cases {
		got := protectReason(c.p, self)
		if got != c.want {
			t.Fatalf("pid %d: got %q want %q", c.p.PID, got, c.want)
		}
	}
}

func TestTagsFor(t *testing.T) {
	tags := tagsFor(rawProc{User: "xhd2015", Cmd: "cloudflared tunnel --url http://127.0.0.1:8422"}, "xhd2015")
	if !contains(tags, "mine") || !contains(tags, "tun") || !contains(tags, "stale") {
		t.Fatalf("%v", tags)
	}
	tags = tagsFor(rawProc{User: "xhd2015", Cmd: "eluc-wt-bin --port 8466"}, "xhd2015")
	if !contains(tags, "stale") {
		t.Fatalf("%v", tags)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
