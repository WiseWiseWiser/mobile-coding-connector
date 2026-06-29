package agentcli

import (
	"errors"
	"net"
	"strings"
	"syscall"
	"testing"
)

func TestPingFailureHintsConnectionRefused(t *testing.T) {
	err := &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
	hints := pingFailureHints("https://agent.example.com", err)
	if len(hints) == 0 {
		t.Fatal("expected hints")
	}
	found := false
	for _, h := range hints {
		if strings.Contains(h, "connection refused") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected connection refused hint, got %v", hints)
	}
}

func TestPingFailureHintsDNS(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "missing.example.com", IsNotFound: true}
	hints := pingFailureHints("https://missing.example.com", err)
	if len(hints) == 0 || !strings.Contains(hints[0], "DNS") {
		t.Fatalf("expected DNS hint, got %v", hints)
	}
}

func TestFormatPingFailureIncludesServer(t *testing.T) {
	out := formatPingFailure("https://agent.example.com", errors.New("boom"))
	if !strings.Contains(out, "https://agent.example.com") {
		t.Fatalf("output missing server URL: %s", out)
	}
	if !strings.Contains(out, "Suggestions:") {
		t.Fatalf("output missing suggestions: %s", out)
	}
}