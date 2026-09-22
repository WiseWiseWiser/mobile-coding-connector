package cloudflare

import (
	"strings"
	"testing"
)

func TestHostPreferredTunnelName(t *testing.T) {
	got := HostPreferredTunnelName("iZ7xv1m5lhf5vzlhgwc4c2Z")
	if got != "ai-critic-iz7xv1m5lhf5vzlhgwc4c2z" {
		t.Fatalf("got %q", got)
	}
	got = HostPreferredTunnelName("Foo.Bar.local")
	if got != "ai-critic-foo" {
		t.Fatalf("got %q", got)
	}
	got = HostPreferredTunnelName("my_host-01")
	if got != "ai-critic-my-host-01" {
		t.Fatalf("got %q", got)
	}
	if !strings.HasPrefix(HostPreferredTunnelName(""), "ai-critic-") {
		t.Fatal("empty hostname")
	}
}

func TestHostPreferredTunnelNameUnique(t *testing.T) {
	a := HostPreferredTunnelNameUnique("box")
	b := HostPreferredTunnelNameUnique("box")
	if !strings.HasPrefix(a, "ai-critic-box-") || len(a) != len("ai-critic-box-")+6 {
		t.Fatalf("a=%q", a)
	}
	if a == b {
		t.Fatal("expected distinct hex suffixes")
	}
}
