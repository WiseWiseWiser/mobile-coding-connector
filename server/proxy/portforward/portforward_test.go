package portforward

import (
	"fmt"
	"sync"
	"testing"
)

func TestAddCloudflareForwardReplacesSameHostnameOnDifferentPort(t *testing.T) {
	m := NewManager()
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)

	if _, err := m.Add(9472, "knowledge-base-782as-sub-server-v2.xhd2015.xyz", ProviderCloudflareOwned); err != nil {
		t.Fatalf("Add first forward error = %v", err)
	}
	if _, err := m.Add(9476, "knowledge-base-782as-sub-server-v2.xhd2015.xyz", ProviderCloudflareOwned); err != nil {
		t.Fatalf("Add replacement forward error = %v", err)
	}

	forwards := m.List()
	if len(forwards) != 1 {
		t.Fatalf("forward count = %d, want 1: %#v", len(forwards), forwards)
	}
	if forwards[0].LocalPort != 9476 {
		t.Fatalf("remaining port = %d, want 9476", forwards[0].LocalPort)
	}
	if provider.StopCount() != 1 {
		t.Fatalf("stale stop count = %d, want 1", provider.StopCount())
	}
}

func TestAddNonCloudflareForwardAllowsDuplicateLabels(t *testing.T) {
	m := NewManager()
	provider := &testProvider{name: ProviderLocaltunnel}
	m.RegisterProvider(provider)

	if _, err := m.Add(3000, "web", ProviderLocaltunnel); err != nil {
		t.Fatalf("Add first forward error = %v", err)
	}
	if _, err := m.Add(3001, "web", ProviderLocaltunnel); err != nil {
		t.Fatalf("Add second forward error = %v", err)
	}

	forwards := m.List()
	if len(forwards) != 2 {
		t.Fatalf("forward count = %d, want 2: %#v", len(forwards), forwards)
	}
	if provider.StopCount() != 0 {
		t.Fatalf("stop count = %d, want 0", provider.StopCount())
	}
}

type testProvider struct {
	name string

	mu    sync.Mutex
	stops int
}

func (p *testProvider) Name() string        { return p.name }
func (p *testProvider) DisplayName() string { return p.name }
func (p *testProvider) Description() string { return p.name }
func (p *testProvider) Available() bool     { return true }

func (p *testProvider) Start(port int, hostname string) (*TunnelHandle, error) {
	resultCh := make(chan TunnelResult, 1)
	resultCh <- TunnelResult{PublicURL: fmt.Sprintf("https://%s", hostname)}
	return &TunnelHandle{
		Result: resultCh,
		Stop: func() {
			p.mu.Lock()
			p.stops++
			p.mu.Unlock()
		},
		Logs: NewLogBuffer(),
	}, nil
}

func (p *testProvider) StopCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stops
}
