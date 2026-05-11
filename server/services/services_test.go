package services

import (
	"fmt"
	"sync"
	"testing"

	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/portforward"
)

func TestEnsurePortForwardReplacesMismatchedForwardForSamePort(t *testing.T) {
	pfm := portforward.NewManager()
	provider := &testPortForwardProvider{name: portforward.ProviderCloudflareOwned}
	pfm.RegisterProvider(provider)

	if _, err := pfm.Add(9476, "knowledge-base-782as-sub-server-v2.xhd2015.xyz", portforward.ProviderCloudflareOwned); err != nil {
		t.Fatalf("Add stale forward error = %v", err)
	}

	m := &Manager{
		processes: map[string]*serviceProcess{
			"svc-v3": {},
		},
		portForwardManager: pfm,
	}

	err := m.ensurePortForward("svc-v3", ServiceDefinition{
		PortForward: &ServicePortForward{
			Port:       9476,
			Provider:   portforward.ProviderCloudflareOwned,
			BaseDomain: "xhd2015.xyz",
			Subdomain:  "knowledge-base-782as-sub-server-v3",
		},
	})
	if err != nil {
		t.Fatalf("ensurePortForward() error = %v", err)
	}

	forwards := pfm.List()
	if len(forwards) != 1 {
		t.Fatalf("forward count = %d, want 1: %#v", len(forwards), forwards)
	}
	got := forwards[0]
	if got.LocalPort != 9476 || got.Label != "knowledge-base-782as-sub-server-v3.xhd2015.xyz" {
		t.Fatalf("forward = port %d label %q, want 9476/v3", got.LocalPort, got.Label)
	}
	if provider.StopCount() != 1 {
		t.Fatalf("stale stop count = %d, want 1", provider.StopCount())
	}
	if !m.processes["svc-v3"].ownedForward {
		t.Fatalf("service did not take ownership of replacement forward")
	}
}

func TestEnsurePortForwardKeepsExistingServiceOwnership(t *testing.T) {
	pfm := portforward.NewManager()
	provider := &testPortForwardProvider{name: portforward.ProviderCloudflareOwned}
	pfm.RegisterProvider(provider)

	if _, err := pfm.Add(9476, "knowledge-base-782as-sub-server-v3.xhd2015.xyz", portforward.ProviderCloudflareOwned); err != nil {
		t.Fatalf("Add existing forward error = %v", err)
	}

	m := &Manager{
		processes: map[string]*serviceProcess{
			"svc-v3": {ownedForward: true},
		},
		portForwardManager: pfm,
	}

	err := m.ensurePortForward("svc-v3", ServiceDefinition{
		PortForward: &ServicePortForward{
			Port:       9476,
			Provider:   portforward.ProviderCloudflareOwned,
			BaseDomain: "xhd2015.xyz",
			Subdomain:  "knowledge-base-782as-sub-server-v3",
		},
	})
	if err != nil {
		t.Fatalf("ensurePortForward() error = %v", err)
	}
	if !m.processes["svc-v3"].ownedForward {
		t.Fatalf("service ownership was cleared for an existing matching forward")
	}
	if provider.StopCount() != 0 {
		t.Fatalf("stop count = %d, want 0", provider.StopCount())
	}
}

type testPortForwardProvider struct {
	name string

	mu    sync.Mutex
	stops int
}

func (p *testPortForwardProvider) Name() string        { return p.name }
func (p *testPortForwardProvider) DisplayName() string { return p.name }
func (p *testPortForwardProvider) Description() string { return p.name }
func (p *testPortForwardProvider) Available() bool     { return true }

func (p *testPortForwardProvider) Start(port int, hostname string) (*portforward.TunnelHandle, error) {
	resultCh := make(chan portforward.TunnelResult, 1)
	resultCh <- portforward.TunnelResult{PublicURL: fmt.Sprintf("https://%s", hostname)}
	return &portforward.TunnelHandle{
		Result: resultCh,
		Stop: func() {
			p.mu.Lock()
			p.stops++
			p.mu.Unlock()
		},
		Logs: portforward.NewLogBuffer(),
	}, nil
}

func (p *testPortForwardProvider) StopCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stops
}
