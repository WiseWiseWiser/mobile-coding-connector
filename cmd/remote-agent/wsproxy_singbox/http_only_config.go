package wsproxy_singbox

import (
	"encoding/json"
	"fmt"
)

const (
	clashAPIListen   = "127.0.0.1:9090"
	webSelectorTag   = "web"
	proxyOutboundTag = "proxy"
	directOutboundTag = "direct"
)

// HttpOnlyConfigOptions configures vpn-http-only sing-box rendering.
type HttpOnlyConfigOptions struct {
	LocalSocksPort  int
	InitialUseProxy bool
	Policy          *DomainPolicy
	DNSHijack       bool
}

func buildHttpOnlyDNSConfigLocal() map[string]any {
	return map[string]any{
		"servers": []map[string]any{
			{"type": "local"},
		},
	}
}

// appendHttpOnlyDNSRouteRules adds DNS hijacking for --dns-hijack.
// With the xray sidecar, resolution uses fakeip (see buildTunDNSConfig) — no resolve action.
func appendHttpOnlyDNSRouteRules(rules []map[string]any) []map[string]any {
	return append(rules,
		map[string]any{
			"type": "logical",
			"mode": "or",
			"rules": []map[string]any{
				{"protocol": "dns"},
				{"port": 53},
			},
			"action": "hijack-dns",
		},
	)
}

func appendBuiltinBypassRules(rules []map[string]any, vmess *VMessParams, localSocksPort int) []map[string]any {
	proxyIPs := resolveHostIPv4CIDRs(vmess.Host)
	rules = append(rules,
		map[string]any{
			"action":   "route",
			"domain":   []string{vmess.Host},
			"outbound": directOutboundTag,
		},
	)
	if len(proxyIPs) > 0 {
		rules = append(rules, map[string]any{
			"action":   "route",
			"ip_cidr":  proxyIPs,
			"outbound": directOutboundTag,
		})
	}
	rules = append(rules, map[string]any{
		"action":   "route",
		"ip_cidr":  lanBypassCIDRs,
		"outbound": directOutboundTag,
	})
	return rules
}

func appendPolicyRouteRules(rules []map[string]any, policy *DomainPolicy) []map[string]any {
	if policy == nil {
		policy = &DomainPolicy{Mode: PolicyBlacklist}
	}

	switch policy.Mode {
	case PolicyWhitelist:
		for _, pattern := range policy.Exclude {
			rules = append(rules, domainRouteRule(pattern, directOutboundTag))
		}
		for _, pattern := range policy.Include {
			rules = append(rules, domainRouteRule(pattern, webSelectorTag))
		}
	default:
		for _, pattern := range policy.Include {
			rules = append(rules, domainRouteRule(pattern, webSelectorTag))
		}
		for _, pattern := range policy.Exclude {
			rules = append(rules, domainRouteRule(pattern, directOutboundTag))
		}
		rules = append(rules, httpOnlyCatchAllRule())
	}
	return rules
}

func httpOnlyCatchAllRule() map[string]any {
	return map[string]any{
		"type": "logical",
		"mode": "or",
		"rules": []map[string]any{
			{"port": []int{80, 443}},
			{"protocol": []string{"http", "tls"}},
		},
		"action":   "route",
		"outbound": webSelectorTag,
	}
}

// BuildSingBoxHttpOnlyTunConfig renders sing-box JSON for vpn-http-only.
func BuildSingBoxHttpOnlyTunConfig(vmess *VMessParams, opts *HttpOnlyConfigOptions) ([]byte, error) {
	if vmess == nil {
		return nil, fmt.Errorf("vmess params required")
	}
	if opts == nil {
		opts = &HttpOnlyConfigOptions{}
	}
	localSocksPort := opts.LocalSocksPort
	if localSocksPort <= 0 {
		return nil, fmt.Errorf("http-only config requires local xray SOCKS port")
	}

	defaultOutbound := directOutboundTag
	if opts.InitialUseProxy {
		defaultOutbound = proxyOutboundTag
	}

	routeRules := []map[string]any{
		{"action": "sniff"},
	}
	if opts.DNSHijack {
		routeRules = appendHttpOnlyDNSRouteRules(routeRules)
	}
	routeRules = appendBuiltinBypassRules(routeRules, vmess, localSocksPort)
	routeRules = appendPolicyRouteRules(routeRules, opts.Policy)

	bindIface := defaultOutboundBindInterface()
	proxyOutbound := map[string]any{
		"type":        "socks",
		"tag":         proxyOutboundTag,
		"server":      "127.0.0.1",
		"server_port": localSocksPort,
		"version":     "5",
		"udp_over_tcp": map[string]any{
			"enabled": true,
			"version": 2,
		},
	}
	directOutbound := map[string]any{
		"type": "direct",
		"tag":  directOutboundTag,
	}
	if bindIface != "" {
		directOutbound["bind_interface"] = bindIface
	}

	dnsCfg := buildHttpOnlyDNSConfigLocal()
	strictRoute := false
	if opts.DNSHijack {
		// fakeip + xray sidecar: DNS resolves via VMess, not direct 8.8.8.8 (often polluted on hotspot).
		dnsCfg = buildTunDNSConfig(vmess.Host, localSocksPort)
		strictRoute = true
	}

	routeCfg := map[string]any{
		"rules":                 routeRules,
		"auto_detect_interface": true,
		"final":                 directOutboundTag,
	}
	if opts.DNSHijack {
		routeCfg["default_domain_resolver"] = "bootstrap"
	}

	cfg := map[string]any{
		"log": map[string]any{
			"level":  "info",
			"output": singBoxLogPath(),
		},
		"dns": dnsCfg,
		"experimental": map[string]any{
			"clash_api": map[string]any{
				"external_controller": clashAPIListen,
			},
		},
		"inbounds": []map[string]any{
			{
				"type":                  "tun",
				"tag":                   "tun-in",
				"address":               []string{"172.19.0.1/30"},
				"mtu":                   1280,
				"auto_route":            true,
				"strict_route":          strictRoute,
				"stack":                 "system",
				"route_exclude_address": tunRouteExcludeAddresses(vmess.Host),
			},
		},
		"outbounds": []map[string]any{
			{
				"type":       "selector",
				"tag":        webSelectorTag,
				"outbounds":  []string{proxyOutboundTag, directOutboundTag},
				"default":    defaultOutbound,
			},
			proxyOutbound,
			directOutbound,
		},
		"route": routeCfg,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal sing-box config: %w", err)
	}
	return data, nil
}