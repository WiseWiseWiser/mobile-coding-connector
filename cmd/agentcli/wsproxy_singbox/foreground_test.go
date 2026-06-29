package wsproxy_singbox

import "testing"

func TestParseTunStartedLine(t *testing.T) {
	line := `INFO inbound/tun[tun-in]: started at utun5`
	iface, ok := parseTunStartedLine(line)
	if !ok {
		t.Fatal("expected match")
	}
	if iface != "utun5" {
		t.Fatalf("iface = %q, want utun5", iface)
	}
}