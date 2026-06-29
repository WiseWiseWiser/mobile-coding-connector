package unified_tunnel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
	"gopkg.in/yaml.v3"
)

func testTunnelManager(t *testing.T) (*UnifiedTunnelManager, string) {
	t.Helper()

	dataDir := t.TempDir()
	oldDataDir := config.DataDir
	config.DataDir = dataDir
	t.Cleanup(func() { config.DataDir = oldDataDir })

	credPath := filepath.Join(dataDir, "tunnel-creds.json")
	if err := os.WriteFile(credPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	cleanupHooks := SetTestProcessHooks(
		func(utm *UnifiedTunnelManager) error {
			utm.running = true
			return nil
		},
		func(utm *UnifiedTunnelManager) {
			utm.running = false
			utm.cmd = nil
		},
	)
	t.Cleanup(cleanupHooks)

	utm := NewUnifiedTunnelManager("test")
	utm.rebuildDebounce = 50 * time.Millisecond
	utm.SetConfig(config.CloudflareTunnelConfig{
		TunnelID:        "7c6e51aa-dcdc-4b7c-b9ae-86ce5d4ec351",
		TunnelName:      "test-extension",
		CredentialsFile: credPath,
	})

	return utm, dataDir
}

func waitForRebuildCount(t *testing.T, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if TestRebuildExecutedCount() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("rebuild count = %d, want %d within %v", TestRebuildExecutedCount(), want, timeout)
}

func readGeneratedConfig(t *testing.T, utm *UnifiedTunnelManager) *CloudflaredConfig {
	t.Helper()
	data, err := os.ReadFile(utm.GetConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg CloudflaredConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return &cfg
}

func hostnamesInConfig(cfg *CloudflaredConfig) []string {
	var hosts []string
	for _, rule := range cfg.Ingress {
		if rule.Hostname != "" {
			hosts = append(hosts, rule.Hostname)
		}
	}
	return hosts
}

// Debounce: three rapid AddMapping calls cause one rebuild.
func TestDebounceCoalescesRapidAddMappings(t *testing.T) {
	utm, _ := testTunnelManager(t)

	mappings := []*IngressMapping{
		{ID: "owned-port-1", Hostname: "alpha.example.com", Service: "http://localhost:1"},
		{ID: "owned-port-2", Hostname: "beta.example.com", Service: "http://localhost:2"},
		{ID: "owned-port-3", Hostname: "gamma.example.com", Service: "http://localhost:3"},
	}
	for _, m := range mappings {
		if err := utm.AddMapping(m); err != nil {
			t.Fatalf("AddMapping(%s): %v", m.ID, err)
		}
	}

	waitForRebuildCount(t, 1, time.Second)

	if got := len(utm.ListMappings()); got != 3 {
		t.Fatalf("len(mappings) = %d, want 3", got)
	}

	cfg := readGeneratedConfig(t, utm)
	hosts := hostnamesInConfig(cfg)
	if len(hosts) != 3 {
		t.Fatalf("ingress hostnames = %v, want 3 entries", hosts)
	}
	for _, want := range []string{"alpha.example.com", "beta.example.com", "gamma.example.com"} {
		if !containsString(hosts, want) {
			t.Fatalf("config missing hostname %q, got %v", want, hosts)
		}
	}
}

// Debounce: unchanged AddMapping does not schedule another rebuild.
func TestDebounceSkipsUnchangedMapping(t *testing.T) {
	utm, _ := testTunnelManager(t)

	mapping := &IngressMapping{
		ID:       "owned-port-9",
		Hostname: "same.example.com",
		Service:  "http://localhost:9",
	}
	if err := utm.AddMapping(mapping); err != nil {
		t.Fatalf("first AddMapping: %v", err)
	}
	waitForRebuildCount(t, 1, time.Second)

	if err := utm.AddMapping(mapping); err != nil {
		t.Fatalf("second AddMapping: %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if got := TestRebuildExecutedCount(); got != 1 {
		t.Fatalf("rebuild count after duplicate add = %d, want 1", got)
	}
}

// Debounce: adds separated by longer than the window trigger separate rebuilds.
func TestDebounceSeparateWindowsTriggerSeparateRebuilds(t *testing.T) {
	utm, _ := testTunnelManager(t)

	if err := utm.AddMapping(&IngressMapping{
		ID: "owned-port-10", Hostname: "one.example.com", Service: "http://localhost:10",
	}); err != nil {
		t.Fatalf("first AddMapping: %v", err)
	}
	waitForRebuildCount(t, 1, time.Second)

	time.Sleep(80 * time.Millisecond)

	if err := utm.AddMapping(&IngressMapping{
		ID: "owned-port-11", Hostname: "two.example.com", Service: "http://localhost:11",
	}); err != nil {
		t.Fatalf("second AddMapping: %v", err)
	}
	waitForRebuildCount(t, 2, time.Second)
}

// RestartMapping bypasses debounce and rebuilds immediately.
func TestRestartMappingBypassesDebounce(t *testing.T) {
	utm, _ := testTunnelManager(t)

	if err := utm.AddMapping(&IngressMapping{
		ID: "owned-port-20", Hostname: "restart.example.com", Service: "http://localhost:20",
	}); err != nil {
		t.Fatalf("AddMapping: %v", err)
	}

	if err := utm.RestartMapping("owned-port-20"); err != nil {
		t.Fatalf("RestartMapping: %v", err)
	}

	waitForRebuildCount(t, 1, time.Second)

	time.Sleep(80 * time.Millisecond)
	if got := TestRebuildExecutedCount(); got != 1 {
		t.Fatalf("rebuild count = %d, want 1 (forced restart cancels pending debounce)", got)
	}
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func TestGeneratedConfigSortedHostnames(t *testing.T) {
	utm, _ := testTunnelManager(t)

	for _, m := range []*IngressMapping{
		{ID: "owned-port-z", Hostname: "zulu.example.com", Service: "http://localhost:26"},
		{ID: "owned-port-a", Hostname: "alpha.example.com", Service: "http://localhost:27"},
	} {
		if err := utm.AddMapping(m); err != nil {
			t.Fatalf("AddMapping: %v", err)
		}
	}
	waitForRebuildCount(t, 1, time.Second)

	cfg := readGeneratedConfig(t, utm)
	raw, err := os.ReadFile(utm.GetConfigPath())
	if err != nil {
		t.Fatalf("read raw config: %v", err)
	}
	text := string(raw)
	alpha := strings.Index(text, "alpha.example.com")
	zulu := strings.Index(text, "zulu.example.com")
	if alpha < 0 || zulu < 0 {
		t.Fatalf("config missing expected hostnames: %s", text)
	}
	if alpha > zulu {
		t.Fatalf("hostnames not sorted in YAML:\n%s", text)
	}
	_ = cfg
}