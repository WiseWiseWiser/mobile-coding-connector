package portforward

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

func useTempMappingFiles(t *testing.T) {
	t.Helper()
	oldReconcile := reconcileOwnedConnectors
	reconcileOwnedConnectors = func() {}
	t.Cleanup(func() { reconcileOwnedConnectors = oldReconcile })
	dir := t.TempDir()
	oldData := config.DataDir
	oldDesired := config.PortMappingDesiredFile
	oldAuth := config.PortMappingAuthFile
	config.DataDir = dir
	config.PortMappingDesiredFile = filepath.Join(dir, "port-mapping-desired.json")
	config.PortMappingAuthFile = filepath.Join(dir, "port-mapping-auth.json")
	t.Cleanup(func() {
		config.DataDir = oldData
		config.PortMappingDesiredFile = oldDesired
		config.PortMappingAuthFile = oldAuth
	})
}

func TestRestoreBackoffDelay(t *testing.T) {
	if got := restoreBackoffDelay(0); got != 0 {
		t.Fatalf("0 failures = %s", got)
	}
	if got := restoreBackoffDelay(1); got != 2*time.Second {
		t.Fatalf("1 = %s, want 2s", got)
	}
	if got := restoreBackoffDelay(2); got != 4*time.Second {
		t.Fatalf("2 = %s, want 4s", got)
	}
	if got := restoreBackoffDelay(5); got != 30*time.Second {
		t.Fatalf("5 = %s, want 30s", got)
	}
	if got := restoreBackoffDelay(20); got != 30*time.Second {
		t.Fatalf("cap = %s, want 30s", got)
	}
}

func TestRestoreDesiredPublishesAuthMapping(t *testing.T) {
	useTempMappingFiles(t)
	m := NewManager()
	m.authSecretPath = filepath.Join(t.TempDir(), "secret")
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)

	if err := config.SetPortMappingDesired(9476, config.PortMappingDesiredSpec{
		Hostname: "eluc-on-mac.xhd2015.xyz",
		Provider: ProviderCloudflareOwned,
	}); err != nil {
		t.Fatal(err)
	}
	if err := config.SetPortMappingAuth(9476, config.PortAuth{
		Required:  true,
		Ephemeral: true,
		TokenMode: config.PortAuthEphemeral,
		Token:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}); err != nil {
		t.Fatal(err)
	}

	m.RestoreDesired()
	t.Cleanup(func() { _ = m.Remove(9476) })

	forwards := m.List()
	if len(forwards) != 1 || forwards[0].LocalPort != 9476 {
		t.Fatalf("forwards = %#v", forwards)
	}
	if forwards[0].Auth == nil || !forwards[0].Auth.Required {
		t.Fatalf("auth = %#v", forwards[0].Auth)
	}
	if provider.LastPort() == 9476 {
		t.Fatal("restored auth mapping tunneled the backend port")
	}
}

func TestRestoreDesiredSkipsNameOnlyPorts(t *testing.T) {
	useTempMappingFiles(t)
	m := NewManager()
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)
	m.RestoreDesired()
	if n := len(m.List()); n != 0 {
		t.Fatalf("restored %d mappings, want 0", n)
	}
}

func TestEnsureDesiredRebuildsDeadHop(t *testing.T) {
	useTempMappingFiles(t)
	m := NewManager()
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)
	if _, err := m.Add(9476, "eluc-on-mac.xhd2015.xyz", ProviderCloudflareOwned); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Remove(9476) })
	m.mu.Lock()
	m.tunnels[9476].hopAlive = func() bool { return false }
	m.mu.Unlock()
	if err := m.ensureDesired(9476, config.PortMappingDesiredSpec{
		Hostname: "eluc-on-mac.xhd2015.xyz",
		Provider: ProviderCloudflareOwned,
	}); err != nil {
		t.Fatal(err)
	}
	if provider.StopCount() != 1 {
		t.Fatalf("dead hop stop count = %d, want 1", provider.StopCount())
	}
	if got := m.List(); len(got) != 1 || got[0].Status != StatusActive {
		t.Fatalf("after rebuild = %#v", got)
	}
}

func TestEnsureDesiredLeavesActiveForward(t *testing.T) {
	useTempMappingFiles(t)
	m := NewManager()
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)
	if _, err := m.Add(9476, "already.xhd2015.xyz", ProviderCloudflareOwned); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Remove(9476) })
	if err := m.ensureDesired(9476, config.PortMappingDesiredSpec{
		Hostname: "other.xhd2015.xyz",
		Provider: ProviderCloudflareOwned,
	}); err != nil {
		t.Fatal(err)
	}
	if got := m.List()[0].Label; got != "already.xhd2015.xyz" {
		t.Fatalf("label = %q, want existing forward kept", got)
	}
}

func TestRestoreTickRespectsBackoff(t *testing.T) {
	useTempMappingFiles(t)
	m := NewManager()
	now := time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	failing := &failProvider{testProvider: testProvider{name: ProviderCloudflareOwned}}
	m.RegisterProvider(failing)

	if err := config.SetPortMappingDesired(9476, config.PortMappingDesiredSpec{
		Hostname: "eluc-on-mac.xhd2015.xyz",
		Provider: ProviderCloudflareOwned,
	}); err != nil {
		t.Fatal(err)
	}

	m.restoreTick()
	if failing.starts != 1 {
		t.Fatalf("starts after first tick = %d", failing.starts)
	}
	m.restoreTick()
	if failing.starts != 1 {
		t.Fatalf("second tick during backoff started again: %d", failing.starts)
	}
	now = now.Add(2 * time.Second)
	m.restoreTick()
	if failing.starts != 2 {
		t.Fatalf("starts after backoff = %d, want 2", failing.starts)
	}
}

type failProvider struct {
	testProvider
	starts int
}

func (p *failProvider) Start(port int, hostname string) (*TunnelHandle, error) {
	p.starts++
	return nil, fmt.Errorf("tunnel boom")
}
