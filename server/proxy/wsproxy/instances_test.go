package wsproxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLegacyConfig(t *testing.T, dir string, cfg *Config) {
	t.Helper()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, configFileName), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationFromLegacyConfig(t *testing.T) {
	dir := t.TempDir()
	SetTestConfigDir(dir)
	defer SetTestConfigDir("")

	writeLegacyConfig(t, dir, &Config{
		UpstreamProxy: "http://127.0.0.1:23456",
		ListenPort:    30001,
		WSPath:        "/ws",
		UUID:          "00000000-0000-4000-8000-000000000001",
		Subdomain:     "ws",
		PublicURL:     "https://ws-abc123.xhd2015.xyz",
		AutoStart:     true,
	})

	instances, err := LoadInstances()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1 (migrated default)", len(instances))
	}
	def := instances[0]
	if def.ID != "default" || !def.Legacy {
		t.Fatalf("default instance = %+v", def)
	}
	if def.BackendType != BackendProxy || def.BackendURL != "http://127.0.0.1:23456" {
		t.Fatalf("backend = %s/%s, want proxy/http://127.0.0.1:23456", def.BackendType, def.BackendURL)
	}
	// Hostname pinned so the upgraded phone keeps working.
	if def.Hostname != "ws-abc123.xhd2015.xyz" {
		t.Fatalf("hostname = %q, want pinned legacy URL host", def.Hostname)
	}
	if def.UUID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("uuid not carried over: %q", def.UUID)
	}
}

func TestMigrationSelfBackendWhenNoUpstream(t *testing.T) {
	dir := t.TempDir()
	SetTestConfigDir(dir)
	defer SetTestConfigDir("")

	// UUID present but no upstream: backend is self (host is the proxy).
	writeLegacyConfig(t, dir, &Config{Subdomain: "ws", UUID: "00000000-0000-4000-8000-000000000002"})

	instances, err := LoadInstances()
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 || instances[0].BackendType != BackendSelf {
		t.Fatalf("instances = %+v, want one self-backend instance", instances)
	}
}

func TestMigrationSkipsNeverStartedConfig(t *testing.T) {
	dir := t.TempDir()
	SetTestConfigDir(dir)
	defer SetTestConfigDir("")

	// No UUID at all = never started: nothing migrates, the store stays empty
	// and the controller keeps reporting "not configured".
	writeLegacyConfig(t, dir, &Config{Subdomain: "ws"})
	empty, err := LoadInstances()
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("unconfigured legacy config must migrate to zero instances, got %+v", empty)
	}
}

func TestValidateInstanceBackends(t *testing.T) {
	inst := &Instance{ID: "a", BackendType: BackendSelf, Subdomain: "a"}
	if err := ValidateInstance(inst); err != nil {
		t.Fatalf("self: %v", err)
	}
	if inst.BackendURL != "" {
		t.Fatalf("self backend must clear url, got %q", inst.BackendURL)
	}
	if inst.UUID == "" || inst.WSPath == "" {
		t.Fatal("uuid/ws_path must be defaulted")
	}

	proxy := &Instance{ID: "b", BackendType: BackendProxy, BackendURL: "10.53.64.69:23456", Subdomain: "b"}
	if err := ValidateInstance(proxy); err != nil {
		t.Fatalf("proxy: %v", err)
	}
	if proxy.BackendURL != "http://10.53.64.69:23456" {
		t.Fatalf("proxy url = %q, want scheme added", proxy.BackendURL)
	}

	bad := &Instance{ID: "c", BackendType: "socks", Subdomain: "c"}
	if err := ValidateInstance(bad); err == nil {
		t.Fatal("unknown backend must be rejected")
	}
}

func TestInstanceAPIFlow(t *testing.T) {
	dir := t.TempDir()
	SetTestConfigDir(dir)
	defer SetTestConfigDir("")

	mux := http.NewServeMux()
	RegisterInstanceAPI(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create a proxy-backend instance.
	body := `{"name":"lab","backend_type":"proxy","backend_url":"http://10.53.64.69:23456","subdomain":"ws-lab"}`
	resp, err := http.Post(srv.URL+"/api/ws-proxy/instances", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Duplicate subdomain rejected.
	resp2, err := http.Post(srv.URL+"/api/ws-proxy/instances", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate subdomain status = %d, want 409", resp2.StatusCode)
	}

	// List shows the created instance (plus migration default if any).
	resp3, err := http.Get(srv.URL + "/api/ws-proxy/instances")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Instances []InstanceWithStatus `json:"instances"`
	}
	_ = json.NewDecoder(resp3.Body).Decode(&list)
	resp3.Body.Close()
	found := false
	for _, inst := range list.Instances {
		// ID defaults to the subdomain when not supplied explicitly.
		if inst.Name == "lab" && inst.ID == "ws-lab" && inst.Backend == "HTTP proxy · http://10.53.64.69:23456" {
			found = true
		}
	}
	if !found {
		t.Fatalf("created instance missing from list: %+v", list.Instances)
	}

	// Delete the non-legacy instance.
	req, _ := http.NewRequest("DELETE", srv.URL+"/api/ws-proxy/instances/ws-lab", nil)
	resp4, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp4.StatusCode)
	}
}
