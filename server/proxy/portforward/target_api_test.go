package portforward

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
)

func setupTargetAPI(t *testing.T) func(string, string, string) *httptest.ResponseRecorder {
	t.Helper()
	useTempMappingFiles(t)
	previous := defaultManager
	defaultManager = NewManager()
	defaultManager.authSecretPath = filepath.Join(t.TempDir(), "secret")
	defaultManager.RegisterProvider(&testProvider{name: ProviderCloudflareOwned})
	t.Cleanup(func() {
		for _, f := range defaultManager.List() {
			_ = defaultManager.RemoveSelector(f.ID)
		}
		defaultManager = previous
	})
	return func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		ctx, cancel := context.WithCancel(r.Context())
		cancel()
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		handlePorts(w, r)
		return w
	}
}
func decodeForward(t *testing.T, w *httptest.ResponseRecorder) PortForward {
	t.Helper()
	if w.Code < 200 || w.Code >= 300 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var f PortForward
	if err := json.Unmarshal(w.Body.Bytes(), &f); err != nil {
		t.Fatal(err)
	}
	return f
}
func TestMappingUpdatePreservesAuthAndPartialTarget(t *testing.T) {
	call := setupTargetAPI(t)
	a := decodeForward(t, call("POST", "/api/ports", `{"port":3080,"label":"mac.example","provider":"cloudflare_owned","target":{"host":"192.168.1.1"},"auth":{"required":true,"token":"fixture-token","tokenMode":"custom"}}`))
	if a.Auth.Token != "" {
		t.Fatal("token leaked in create response")
	}
	b := decodeForward(t, call("PUT", "/api/ports?id="+a.ID, `{"target":{"host":"macbook.local"}}`))
	if b.ID != a.ID || b.Target.Port != 3080 || b.Target.Scheme != "http" || b.Target.Host != "macbook.local" || b.Label != "mac.example" || !b.Auth.Required {
		t.Fatalf("omitted field reset: %+v", b)
	}
	store, err := config.LoadMappings()
	if err != nil {
		t.Fatal(err)
	}
	if store.Mappings[a.ID].Auth.Token != "fixture-token" {
		t.Fatal("auth rotated during target-only update")
	}
	before := defaultManager.List()[0]
	w := call("PUT", "/api/ports?id="+a.ID, `{"target":{"host":"http://invalid/path"}}`)
	if w.Code != 400 {
		t.Fatalf("invalid update %d", w.Code)
	}
	after := defaultManager.List()[0]
	if before.Target.Host != after.Target.Host {
		t.Fatal("invalid update stopped old target")
	}
	w = call("DELETE", "/api/ports?id="+a.ID, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	defaultManager.RestoreDesired()
	if len(defaultManager.List()) != 0 {
		t.Fatal("deleted mapping restored")
	}
}
func TestAmbiguousPortDoesNotDeleteOrUpdateEitherMapping(t *testing.T) {
	call := setupTargetAPI(t)
	local := decodeForward(t, call("POST", "/api/ports", `{"port":3080,"label":"pi.example","provider":"cloudflare_owned"}`))
	remote := decodeForward(t, call("POST", "/api/ports", `{"port":3080,"label":"mac.example","provider":"cloudflare_owned","target":{"host":"192.168.1.1"}}`))
	if local.ID == remote.ID {
		t.Fatal("ID collision")
	}
	for _, method := range []string{"DELETE", "PUT"} {
		w := call(method, "/api/ports?port=3080", `{"label":"oops.example"}`)
		if w.Code != 409 || !strings.Contains(w.Body.String(), "matches") {
			t.Fatalf("%s: %d %s", method, w.Code, w.Body.String())
		}
	}
	if len(defaultManager.List()) != 2 {
		t.Fatal("ambiguous mutation deleted mapping")
	}
}
func TestLegacyAuthAndNamesAdaptersAreIDIsolated(t *testing.T) {
	call := setupTargetAPI(t)
	f := decodeForward(t, call("POST", "/api/ports", `{"port":3080,"label":"mac.example","provider":"cloudflare_owned","target":{"host":"192.168.1.1"},"auth":{"required":true,"token":"mac-only"}}`))
	w := httptest.NewRecorder()
	handlePortMappingAuth(w, httptest.NewRequest(http.MethodGet, "/api/ports/mapping-auth", nil))
	var auth map[string]config.PortAuth
	if err := json.Unmarshal(w.Body.Bytes(), &auth); err != nil {
		t.Fatal(err)
	}
	if auth[f.ID].Token != "mac-only" {
		t.Fatal("explicit credential reader lost ID")
	}
	if _, ok := auth["3080"]; ok {
		t.Fatal("network auth exported under a port alias")
	}
	w = httptest.NewRecorder()
	handleSavedMappings(w, httptest.NewRequest("GET", "/api/ports/mappings", nil))
	if strings.Contains(w.Body.String(), "mac-only") {
		t.Fatal("saved mapping listing leaked token")
	}
}
