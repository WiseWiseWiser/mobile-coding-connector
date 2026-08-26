package wsproxy_singbox

import (
	"strings"
	"testing"
)

func TestParseAlsoProxyPatternsOK(t *testing.T) {
	patterns, err := ParseAlsoProxyPatterns([]string{
		"git.example.com:22",
		"*.db.internal:6606",
		"git.example.com",
		"*.example.com",
	})
	if err != nil {
		t.Fatalf("ParseAlsoProxyPatterns: %v", err)
	}
	if len(patterns) != 4 {
		t.Fatalf("len=%d want 4", len(patterns))
	}

	if patterns[0].Host != "git.example.com" || patterns[0].Port != 22 || patterns[0].Wildcard {
		t.Fatalf("patterns[0]=%+v", patterns[0])
	}
	if patterns[1].Host != ".db.internal" || patterns[1].Port != 6606 || !patterns[1].Wildcard {
		t.Fatalf("patterns[1]=%+v", patterns[1])
	}
	if patterns[2].Host != "git.example.com" || patterns[2].Port != 0 {
		t.Fatalf("patterns[2]=%+v", patterns[2])
	}
	if patterns[3].Host != ".example.com" || patterns[3].Port != 0 || !patterns[3].Wildcard {
		t.Fatalf("patterns[3]=%+v", patterns[3])
	}
}

func TestParseAlsoProxyPatternPortOnlyRejected(t *testing.T) {
	_, err := ParseAlsoProxyPatterns([]string{":22"})
	if err == nil {
		t.Fatal("expected error for :22")
	}
	if !strings.Contains(err.Error(), "port-only") {
		t.Fatalf("err=%v, want port-only message", err)
	}
}

func TestParseAlsoProxyPatternInvalidPort(t *testing.T) {
	for _, raw := range []string{"host:", "host:abc", "host:0", "host:65536"} {
		_, err := ParseAlsoProxyPatterns([]string{raw})
		if err == nil {
			t.Fatalf("%q: expected error", raw)
		}
	}
}

func TestAlsoProxyRouteRule(t *testing.T) {
	withPort := alsoProxyRouteRule(AlsoProxyPattern{
		Wildcard: false, Host: "git.example.com", Port: 22,
	}, webSelectorTag)
	if withPort["domain"].([]string)[0] != "git.example.com" {
		t.Fatalf("domain=%v", withPort["domain"])
	}
	if withPort["port"] != 22 {
		t.Fatalf("port=%v", withPort["port"])
	}
	if withPort["outbound"] != webSelectorTag {
		t.Fatalf("outbound=%v", withPort["outbound"])
	}

	wildcardAll := alsoProxyRouteRule(AlsoProxyPattern{
		Wildcard: true, Host: ".example.com", Port: 0,
	}, webSelectorTag)
	if _, ok := wildcardAll["port"]; ok {
		t.Fatalf("all-ports rule should omit port: %v", wildcardAll)
	}
	if wildcardAll["domain_suffix"].([]string)[0] != ".example.com" {
		t.Fatalf("domain_suffix=%v", wildcardAll["domain_suffix"])
	}
}
