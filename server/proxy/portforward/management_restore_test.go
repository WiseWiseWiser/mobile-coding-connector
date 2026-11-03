package portforward

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/domains"
)

func TestRestoredManagementMappingPreservesDirectLoopbackCredentials(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer management-fixture" {
			http.Error(w, "unauthorized", 401)
			return
		}
		io.WriteString(w, "management-ok")
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)
	port, _ := strconv.Atoi(u.Port())
	previous := domains.GetServerPort()
	domains.SetServerPort(port)
	defer domains.SetServerPort(previous)
	rec := config.MappingRecord{ID: config.LocalMappingID(port), Target: config.OriginTarget{Scheme: "http", Host: "127.0.0.1", Port: port}, Hostname: "agent.example", Provider: ProviderCloudflareOwned, Desired: true}
	m := NewManager()
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)
	if err := m.ensureRecord(rec); err != nil {
		t.Fatal(err)
	}
	defer m.RemoveSelector(rec.ID)
	if provider.LastPort() != port {
		t.Fatalf("management restored through credential-stripping origin hop: got %d want %d", provider.LastPort(), port)
	}
	request, _ := http.NewRequest("GET", fmtHop(provider.LastPort()), nil)
	request.Header.Set("Authorization", "Bearer management-fixture")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("management token rejected after restore: %s", response.Status)
	}
}

func TestManagementDirectPathExceptionNeverAppliesToOtherOrigins(t *testing.T) {
	previous := domains.GetServerPort()
	domains.SetServerPort(23712)
	defer domains.SetServerPort(previous)
	for _, tc := range []struct {
		name   string
		target config.OriginTarget
		auth   config.PortAuth
	}{
		{"network-same-port", config.OriginTarget{Scheme: "http", Host: "192.168.1.1", Port: 23712}, config.PortAuth{}},
		{"scoped-ipv6", config.OriginTarget{Scheme: "http", Host: "fe80::1234%eth0", Port: 23712}, config.PortAuth{}},
		{"other-local-app", config.OriginTarget{Scheme: "http", Host: "127.0.0.1", Port: 3080}, config.PortAuth{}},
		{"https", config.OriginTarget{Scheme: "https", Host: "127.0.0.1", Port: 23712}, config.PortAuth{}},
		{"auth-gate", config.OriginTarget{Scheme: "http", Host: "127.0.0.1", Port: 23712}, config.PortAuth{Required: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := recordForward(config.MappingRecord{Target: tc.target, Auth: tc.auth}); got.Target == nil {
				t.Fatal("origin credential isolation bypassed")
			}
		})
	}
}
