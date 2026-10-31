package domains

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
)

func TestSavingDomainsPreservesSharedTunnelName(t *testing.T) {
	oldPath := getDomainsFile()
	SetDomainsFile(filepath.Join(t.TempDir(), "domains.json"))
	t.Cleanup(func() { SetDomainsFile(oldPath) })
	if err := SaveDomains(&DomainsConfig{TunnelName: "existing-shared-tunnel"}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/domains", strings.NewReader(`{"domains":[{"domain":"bootstrap.example","provider":"cloudflare"}]}`))
	handleSaveDomains(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("save HTTP %d", w.Code)
	}
	cfg, err := LoadDomains()
	if err != nil || cfg.TunnelName != "existing-shared-tunnel" {
		t.Fatalf("identity changed: %+v %v", cfg, err)
	}
}

func TestDomainStopAPIOnlyTargetsConfiguredBootstrap(t *testing.T) {
	oldPath, oldController := getDomainsFile(), bootstrapDomains
	SetDomainsFile(filepath.Join(t.TempDir(), "domains.json"))
	r := fakeReconciler(t)
	bootstrapDomains = r
	t.Cleanup(func() { bootstrapDomains = oldController; SetDomainsFile(oldPath) })
	if err := SaveDomains(&DomainsConfig{Domains: []DomainEntry{{Domain: "bootstrap.example", Provider: ProviderCloudflare}}}); err != nil {
		t.Fatal(err)
	}
	var stops atomic.Int32
	r.stop = func(domain string, _ cloudflareSettings.LogFunc) error {
		if domain != "bootstrap.example" {
			t.Errorf("stopped other hostname %q", domain)
		}
		stops.Add(1)
		return nil
	}
	call := func(domain string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/domains/tunnel/stop", strings.NewReader(`{"domain":"`+domain+`"}`))
		handleTunnelStop(w, req)
		return w.Code
	}
	if code := call("unrelated.example"); code != http.StatusNotFound {
		t.Fatalf("unconfigured stop HTTP %d", code)
	}
	if stops.Load() != 0 {
		t.Fatal("unconfigured host affected the backend")
	}
	if code := call("bootstrap.example"); code != http.StatusOK {
		t.Fatalf("configured stop HTTP %d", code)
	}
	if stops.Load() != 1 {
		t.Fatalf("stops=%d", stops.Load())
	}
	if status, _ := r.status("bootstrap.example"); status.Status != "stopped" {
		t.Fatalf("status=%+v", status)
	}
}
