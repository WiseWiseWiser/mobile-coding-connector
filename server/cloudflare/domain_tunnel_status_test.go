package cloudflare

import (
	"testing"
)

func TestGetDomainTunnelStatusUsesEdgeDialsForProxySessions(t *testing.T) {
	SetTestProxySession("app.example.com", true)
	t.Cleanup(func() { SetTestProxySession("app.example.com", false) })

	restore := SetTestProxyDialCounts(map[string]int{"app.example.com": 12}, nil)
	t.Cleanup(restore)

	status := GetDomainTunnelStatus("app.example.com")
	if status.Status != "active" {
		t.Fatalf("status = %#v, want active when the edge reports dials", status)
	}

	restoreDead := SetTestProxyDialCounts(map[string]int{"app.example.com": 0}, nil)
	t.Cleanup(restoreDead)

	status = GetDomainTunnelStatus("app.example.com")
	if status.Status != "error" || status.Error != "no live dial pool on the edge" {
		t.Fatalf("status = %#v, want error when the edge reports 0 dials", status)
	}
}

func TestHostDomainStatusRequiresEdgeRegistration(t *testing.T) {
	const domain = "mac-agent-aes42.example.com"

	stopped := hostDomainStatus(domain, false, false, 0)
	if stopped.Status != "connecting" {
		t.Fatalf("not running: %#v", stopped)
	}

	connecting := hostDomainStatus(domain, true, false, 0)
	if connecting.Status != "connecting" {
		t.Fatalf("no probe yet: %#v", connecting)
	}

	down := hostDomainStatus(domain, true, true, 530)
	if down.Status != "error" || down.Error != "no live edge connector" {
		t.Fatalf("530: %#v", down)
	}

	ok := hostDomainStatus(domain, true, true, 200)
	if ok.Status != "active" {
		t.Fatalf("200: %#v", ok)
	}
}

func TestGetDomainTunnelStatusKeepsOptimisticWhenEdgeUnreachable(t *testing.T) {
	SetTestProxySession("app.example.com", true)
	t.Cleanup(func() { SetTestProxySession("app.example.com", false) })

	restore := SetTestProxyDialCounts(nil, errTestEdgeDown)
	t.Cleanup(restore)

	status := GetDomainTunnelStatus("app.example.com")
	if status.Status != "active" {
		t.Fatalf("status = %#v, want the old optimistic answer when the edge cannot be reached", status)
	}
}

type testEdgeDownError struct{}

func (testEdgeDownError) Error() string { return "edge down" }

var errTestEdgeDown = testEdgeDownError{}
