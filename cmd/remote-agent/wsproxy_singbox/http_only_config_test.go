package wsproxy_singbox

import (
	"encoding/json"
	"testing"
)

func TestBuildSingBoxHttpOnlyTunConfigDefaults(t *testing.T) {
	vmess := &VMessParams{
		Host: "proxy.example.com",
		Port: "443",
		UUID: "11111111-2222-4333-8444-555555555555",
		Path: "/ws",
		TLS:  "tls",
	}
	data, err := BuildSingBoxHttpOnlyTunConfig(vmess, &HttpOnlyConfigOptions{
		LocalSocksPort:  11080,
		InitialUseProxy: true,
	})
	if err != nil {
		t.Fatalf("BuildSingBoxHttpOnlyTunConfig: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	route, _ := cfg["route"].(map[string]any)
	if route["final"] != directOutboundTag {
		t.Fatalf("final = %v, want %s", route["final"], directOutboundTag)
	}
	if _, ok := route["default_domain_resolver"]; ok {
		t.Fatalf("default_domain_resolver should be omitted without --dns-hijack")
	}
	rules := routeRulesFromCfg(cfg)
	if rulesContainHijackDNS(rules) {
		t.Fatalf("hijack-dns should be off by default: %v", rules)
	}
	dns, _ := cfg["dns"].(map[string]any)
	servers, _ := dns["servers"].([]any)
	if len(servers) != 1 {
		t.Fatalf("dns servers = %v, want local only", servers)
	}
}

func TestBuildSingBoxHttpOnlyTunConfigDNSHijack(t *testing.T) {
	vmess := &VMessParams{Host: "proxy.example.com", Port: "443", UUID: "u", Path: "/ws", TLS: "tls"}
	data, err := BuildSingBoxHttpOnlyTunConfig(vmess, &HttpOnlyConfigOptions{
		LocalSocksPort:  11080,
		InitialUseProxy: true,
		DNSHijack:       true,
	})
	if err != nil {
		t.Fatalf("BuildSingBoxHttpOnlyTunConfig: %v", err)
	}
	var cfg map[string]any
	_ = json.Unmarshal(data, &cfg)
	rules := routeRulesFromCfg(cfg)
	if !rulesContainHijackDNS(rules) {
		t.Fatalf("missing hijack-dns rule: %v", rules)
	}
	route, _ := cfg["route"].(map[string]any)
	if route["default_domain_resolver"] != "bootstrap" {
		t.Fatalf("default_domain_resolver = %v, want bootstrap", route["default_domain_resolver"])
	}
	dns, _ := cfg["dns"].(map[string]any)
	servers, _ := dns["servers"].([]any)
	foundFakeIP := false
	for _, s := range servers {
		server, _ := s.(map[string]any)
		if server["type"] == "fakeip" {
			foundFakeIP = true
		}
	}
	if !foundFakeIP {
		t.Fatalf("dns servers = %v, want fakeip for --dns-hijack", servers)
	}
	tun := findTunInbound(cfg)
	if tun["strict_route"] != true {
		t.Fatalf("strict_route = %v, want true with --dns-hijack", tun["strict_route"])
	}
}

func TestBuildSingBoxHttpOnlyWhitelistOmitsCatchAll(t *testing.T) {
	vmess := &VMessParams{Host: "proxy.example.com", Port: "443", UUID: "u", Path: "/ws", TLS: "tls"}
	policy, err := ParseDomainPolicy(PolicyInput{Include: []string{"*.corp.com"}})
	if err != nil {
		t.Fatalf("ParseDomainPolicy: %v", err)
	}
	data, err := BuildSingBoxHttpOnlyTunConfig(vmess, &HttpOnlyConfigOptions{
		LocalSocksPort: 11080,
		Policy:         policy,
	})
	if err != nil {
		t.Fatalf("BuildSingBoxHttpOnlyTunConfig: %v", err)
	}
	var cfg map[string]any
	_ = json.Unmarshal(data, &cfg)
	rules := routeRulesFromCfg(cfg)
	if rulesContainCatchAll(rules) {
		t.Fatalf("whitelist should not include catch-all: %v", rules)
	}
}

func findTunInbound(cfg map[string]any) map[string]any {
	inbounds, _ := cfg["inbounds"].([]any)
	for _, in := range inbounds {
		m, _ := in.(map[string]any)
		if m != nil && m["type"] == "tun" {
			return m
		}
	}
	return nil
}

func findOutbound(cfg map[string]any, typ string) map[string]any {
	outbounds, _ := cfg["outbounds"].([]any)
	for _, out := range outbounds {
		m, _ := out.(map[string]any)
		if m != nil && m["type"] == typ {
			return m
		}
	}
	return nil
}

func routeRulesFromCfg(cfg map[string]any) []map[string]any {
	route, _ := cfg["route"].(map[string]any)
	rules, _ := route["rules"].([]any)
	var out []map[string]any
	for _, r := range rules {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func rulesContainHijackDNS(rules []map[string]any) bool {
	for _, r := range rules {
		if r["action"] == "hijack-dns" {
			return true
		}
	}
	return false
}

func rulesContainCatchAll(rules []map[string]any) bool {
	for _, r := range rules {
		if r["type"] == "logical" && r["outbound"] == webSelectorTag {
			return true
		}
	}
	return false
}