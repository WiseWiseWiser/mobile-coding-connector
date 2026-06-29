package wsproxy_singbox

const (
	clashAPIListen   = "127.0.0.1:9090"
	webSelectorTag   = "web"
	proxyOutboundTag = "proxy"
	directOutboundTag = "direct"
)

// HttpOnlyConfigOptions configures vpn --http-only sing-box rendering.
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

func appendPolicyRouteRules(rules []map[string]any, policy *DomainPolicy, proxyOutbound string, httpOnly bool) []map[string]any {
	if policy == nil {
		if httpOnly {
			policy = &DomainPolicy{Mode: PolicyBlacklist}
		} else {
			return rules
		}
	}
	if len(policy.Include) == 0 && len(policy.Exclude) == 0 && !httpOnly {
		return rules
	}

	switch policy.Mode {
	case PolicyWhitelist:
		for _, pattern := range policy.Exclude {
			rules = append(rules, domainRouteRule(pattern, directOutboundTag))
		}
		for _, pattern := range policy.Include {
			rules = append(rules, domainRouteRule(pattern, proxyOutbound))
		}
	default:
		for _, pattern := range policy.Include {
			rules = append(rules, domainRouteRule(pattern, proxyOutbound))
		}
		for _, pattern := range policy.Exclude {
			rules = append(rules, domainRouteRule(pattern, directOutboundTag))
		}
		if httpOnly {
			rules = append(rules, httpOnlyCatchAllRule())
		}
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

// BuildSingBoxHttpOnlyTunConfig renders sing-box JSON for --http-only mode.
func BuildSingBoxHttpOnlyTunConfig(vmess *VMessParams, opts *HttpOnlyConfigOptions) ([]byte, error) {
	if opts == nil {
		opts = &HttpOnlyConfigOptions{}
	}
	return BuildSingBoxTunConfig(vmess, &BuildConfigOptions{
		LocalSocksPort:  opts.LocalSocksPort,
		HttpOnly:        true,
		Policy:          opts.Policy,
		DNSHijack:       opts.DNSHijack,
		InitialUseProxy: opts.InitialUseProxy,
	})
}