package wsproxy_singbox

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestDefaultLookupHostIPv4RespectsTimeout(t *testing.T) {
	start := time.Now()
	ips := defaultLookupHostIPv4("definitely-does-not-exist-xyz.invalid")
	elapsed := time.Since(start)
	if elapsed > hostLookupTimeout+time.Second {
		t.Fatalf("lookup took %s, want <= %s", elapsed, hostLookupTimeout)
	}
	if len(ips) != 0 {
		t.Fatalf("ips = %#v, want nil", ips)
	}
}

func TestResolveHostIPv4CIDRsFromIPs(t *testing.T) {
	old := lookupHostIPv4
	defer func() { lookupHostIPv4 = old }()

	lookupHostIPv4 = func(host string) []net.IP {
		return []net.IP{
			net.ParseIP("1.2.3.4"),
			net.ParseIP("1.2.3.4"),
			net.ParseIP("2001:db8::1"),
		}
	}

	cidrs := resolveHostIPv4CIDRs("proxy.example.com")
	if len(cidrs) != 1 || cidrs[0] != "1.2.3.4/32" {
		t.Fatalf("cidrs = %#v, want [1.2.3.4/32]", cidrs)
	}
}

func TestProxyServerAddressUsesResolvedIPv4(t *testing.T) {
	if got := proxyServerAddress("proxy.example.com", []string{"1.2.3.4/32", "5.6.7.8/32"}); got != "1.2.3.4" {
		t.Fatalf("proxyServerAddress = %q, want 1.2.3.4", got)
	}
	if got := proxyServerAddress("proxy.example.com", nil); got != "proxy.example.com" {
		t.Fatalf("proxyServerAddress = %q, want hostname fallback", got)
	}
}

func TestBuildSingBoxTunConfigUsesBootstrapDNSForProxyHost(t *testing.T) {
	old := lookupHostIPv4
	defer func() { lookupHostIPv4 = old }()

	lookupHostIPv4 = func(host string) []net.IP {
		return []net.IP{net.ParseIP("93.184.216.34")}
	}

	vmess := &VMessParams{
		Host: "proxy.example.com",
		Port: "443",
		UUID: "11111111-2222-4333-8444-555555555555",
		Path: "/ws",
		TLS:  "tls",
	}
	data, err := BuildSingBoxTunConfig(vmess, nil)
	if err != nil {
		t.Fatalf("BuildSingBoxTunConfig: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	outbounds, _ := cfg["outbounds"].([]any)
	if len(outbounds) == 0 {
		t.Fatal("missing outbounds")
	}
	proxy, _ := outbounds[0].(map[string]any)
	if proxy["server"] != "93.184.216.34" {
		t.Fatalf("proxy server = %v, want resolved IPv4", proxy["server"])
	}
	if _, ok := proxy["domain_resolver"]; ok {
		t.Fatalf("domain_resolver should be omitted when server is IPv4: %v", proxy)
	}
	dns, _ := cfg["dns"].(map[string]any)
	servers, _ := dns["servers"].([]any)
	var bootstrap map[string]any
	for _, s := range servers {
		server, _ := s.(map[string]any)
		if server["tag"] == "bootstrap" {
			bootstrap = server
			break
		}
	}
	if bootstrap == nil {
		t.Fatalf("missing bootstrap DNS server: %v", servers)
	}
	if bootstrap["detour"] != "direct" {
		t.Fatalf("bootstrap detour = %v, want direct", bootstrap["detour"])
	}
	rules, _ := dns["rules"].([]any)
	rule0, _ := rules[0].(map[string]any)
	if rule0["server"] != "bootstrap" {
		t.Fatalf("proxy-host DNS rule server = %v, want bootstrap", rule0["server"])
	}
}

func TestBuildSingBoxTunConfigRemoteDNSDetour(t *testing.T) {
	vmess := &VMessParams{
		Host: "proxy.example.com",
		Port: "443",
		UUID: "11111111-2222-4333-8444-555555555555",
		Path: "/ws",
		TLS:  "tls",
	}
	t.Run("native vmess", func(t *testing.T) {
		data, err := BuildSingBoxTunConfig(vmess, nil)
		if err != nil {
			t.Fatalf("BuildSingBoxTunConfig: %v", err)
		}
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		dns, _ := cfg["dns"].(map[string]any)
		servers, _ := dns["servers"].([]any)
		for _, s := range servers {
			server, _ := s.(map[string]any)
			if server["tag"] == "remote" {
				if server["type"] != "udp" {
					t.Fatalf("remote DNS type = %v, want udp", server["type"])
				}
				if server["detour"] != "direct" {
					t.Fatalf("remote DNS detour = %v, want direct", server["detour"])
				}
				return
			}
		}
		t.Fatal("missing remote DNS server")
	})

	t.Run("xray sidecar uses fakeip only", func(t *testing.T) {
		data, err := BuildSingBoxTunConfig(vmess, &BuildConfigOptions{LocalSocksPort: 1080})
		if err != nil {
			t.Fatalf("BuildSingBoxTunConfig: %v", err)
		}
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		dns, _ := cfg["dns"].(map[string]any)
		servers, _ := dns["servers"].([]any)
		for _, s := range servers {
			server, _ := s.(map[string]any)
			if server["tag"] == "remote" {
				t.Fatalf("sidecar config should not include remote DNS server: %v", servers)
			}
		}
		route, _ := cfg["route"].(map[string]any)
		if route["default_domain_resolver"] != "bootstrap" {
			t.Fatalf("default_domain_resolver = %v, want bootstrap", route["default_domain_resolver"])
		}
	})
}

func TestResolveHostIPv4CIDRsLookupFailure(t *testing.T) {
	old := lookupHostIPv4
	defer func() { lookupHostIPv4 = old }()

	lookupHostIPv4 = func(host string) []net.IP {
		return nil
	}

	if cidrs := resolveHostIPv4CIDRs("missing.example.com"); len(cidrs) != 0 {
		t.Fatalf("cidrs = %#v, want nil", cidrs)
	}
}