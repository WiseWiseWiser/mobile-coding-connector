package systemservices

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/proxy/wsproxy"
	"github.com/xhd2015/ai-critic/server/services"
)

func TestRegisterAddsEverySystemServiceOnce(t *testing.T) {
	m := services.NewManagerFromDefinitions(nil)

	if err := Register(m); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := Register(m); err != nil {
		t.Fatalf("second Register() must be a no-op, got error = %v", err)
	}

	registered := m.SystemServices()
	if len(registered) != 4 {
		t.Fatalf("len(SystemServices()) = %d, want 4: %#v", len(registered), registered)
	}

	want := []string{WSProxyID, CFProxyID, GoModID, OpenClawID}
	for i, id := range want {
		if registered[i].ID != id {
			t.Fatalf("SystemServices()[%d].ID = %q, want %q", i, registered[i].ID, id)
		}
		if registered[i].Name == "" || registered[i].Description == "" {
			t.Fatalf("system service %s is missing display fields: %#v", id, registered[i])
		}
	}
}

func TestRegisteredSystemServicesAreListedAsSystem(t *testing.T) {
	stubEdge(t, map[string]int{}, nil)
	m := services.NewManagerFromDefinitions(nil)
	if err := Register(m); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	seen := 0
	for _, status := range m.List() {
		if status.Kind != services.ServiceKindSystem {
			t.Fatalf("status %s kind = %q, want system", status.ID, status.Kind)
		}
		if status.Command != "" {
			t.Fatalf("system service %s exposed a command %q", status.ID, status.Command)
		}
		seen++
	}
	if seen != 4 {
		t.Fatalf("listed %d system services, want 4", seen)
	}
}

// useTempWSProxyConfig points ws-proxy at an empty directory for the test.
func useTempWSProxyConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wsproxy.SetTestConfigDir(dir)
	t.Cleanup(func() { wsproxy.SetTestConfigDir("") })
	return dir
}

func TestWSProxyUnconfiguredHidesAutoStartAndPublicURL(t *testing.T) {
	useTempWSProxyConfig(t)

	runtime := wsProxyController{}.SystemStatus()

	if runtime.Running {
		t.Fatal("ws-proxy must not report running without a config")
	}
	if runtime.AutoStart != nil {
		t.Fatalf("AutoStart = %v, want nil so the UI hides enable/disable", *runtime.AutoStart)
	}
	// The derived URL would embed a freshly generated random instance id.
	if runtime.PublicURL != "" {
		t.Fatalf("PublicURL = %q, want empty for an unconfigured proxy", runtime.PublicURL)
	}
	if runtime.Detail != "not configured" {
		t.Fatalf("Detail = %q, want %q", runtime.Detail, "not configured")
	}

	err := wsProxyController{}.SetSystemAutoStart(true)
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("SetSystemAutoStart() error = %v, want a not-configured error", err)
	}
	// Refusing must not create a config file.
	if wsproxy.ConfigExists() {
		t.Fatal("SetSystemAutoStart wrote a ws-proxy config it should have refused")
	}
}

func TestWSProxyConfiguredReportsAutoStartAndStableURL(t *testing.T) {
	dir := useTempWSProxyConfig(t)

	cfg := wsproxy.Config{
		UpstreamProxy: "http://127.0.0.1:7890",
		ListenPort:    10808,
		WSPath:        "/ws",
		Subdomain:     "ws",
		InstanceID:    "abc123",
		AutoStart:     true,
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ws-proxy.json"), encoded, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	runtime := wsProxyController{}.SystemStatus()

	if runtime.AutoStart == nil {
		t.Fatal("AutoStart = nil, want the value from the saved config")
	}
	if !*runtime.AutoStart {
		t.Fatal("AutoStart = false, want true from the saved config")
	}
	if runtime.Running {
		t.Fatal("ws-proxy must not report running when nothing is listening")
	}
	if runtime.Detail != "not running" {
		t.Fatalf("Detail = %q, want %q", runtime.Detail, "not running")
	}

	// The saved instance id makes the derived hostname stable across calls.
	first := wsProxyController{}.SystemStatus().PublicURL
	second := wsProxyController{}.SystemStatus().PublicURL
	if first != second {
		t.Fatalf("PublicURL is not stable: %q then %q", first, second)
	}
	if first != "" && !strings.Contains(first, "abc123") {
		t.Fatalf("PublicURL = %q, want it to use the saved instance id", first)
	}

	// Disabling keeps the saved instance id instead of freezing a random one.
	if err := (wsProxyController{}).SetSystemAutoStart(false); err != nil {
		t.Fatalf("SetSystemAutoStart(false) error = %v", err)
	}
	reloaded, err := wsproxy.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if reloaded.InstanceID != "abc123" {
		t.Fatalf("InstanceID = %q after toggle, want abc123 preserved", reloaded.InstanceID)
	}
	if reloaded.AutoStart {
		t.Fatal("AutoStart = true after disabling")
	}
}
