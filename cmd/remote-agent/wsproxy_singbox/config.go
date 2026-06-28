package wsproxy_singbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	remoteDNSAddr    = "8.8.8.8"
	bootstrapDNSAddr = "8.8.8.8"
	fakeIPRange      = "198.18.0.0/15"
	tunDNSAddress    = "172.19.0.2" // client DNS on 172.19.0.1/30 TUN subnet
)

var lanBypassCIDRs = []string{
	"127.0.0.0/8",
	"192.168.0.0/16",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"17.0.0.0/8",
}

// tunExcludedCIDRs bypasses the TUN at the OS level. Keep 172.16.0.0/12 out of
// this list so hotspot gateways like 172.20.10.1 still enter the TUN and DNS
// hijack can intercept port-53 queries instead of using polluted router DNS.
var tunExcludedCIDRs = []string{
	"192.168.0.0/16",
	"10.0.0.0/8",
}

// Public DNS resolvers used by the bootstrap server must bypass the TUN at the
// OS level so xray/sing-box can resolve the ws-proxy host before the tunnel is up.
var bootstrapExcludeCIDRs = []string{
	"8.8.8.8/32",
	"1.1.1.1/32",
}

const hostLookupTimeout = 3 * time.Second

var bootstrapResolvers = []string{"1.1.1.1:53", "8.8.8.8:53"}

// lookupHostIPv4 is overridden in tests.
var lookupHostIPv4 = defaultLookupHostIPv4

func defaultLookupHostIPv4(host string) []net.IP {
	ctx, cancel := context.WithTimeout(context.Background(), hostLookupTimeout)
	defer cancel()

	ips, _ := lookupHostIPv4WithResolver(ctx, net.DefaultResolver, host)
	if len(ips) > 0 {
		return ips
	}
	for _, resolver := range bootstrapResolvers {
		ips, _ = lookupHostIPv4ViaUDP(ctx, resolver, host)
		if len(ips) > 0 {
			return ips
		}
	}
	return nil
}

func lookupHostIPv4WithResolver(ctx context.Context, resolver *net.Resolver, host string) ([]net.IP, error) {
	ips, err := resolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func lookupHostIPv4ViaUDP(ctx context.Context, resolverAddr, host string) ([]net.IP, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", resolverAddr)
		},
	}
	return lookupHostIPv4WithResolver(ctx, r, host)
}

func resolveHostIPv4CIDRs(host string) []string {
	ips := lookupHostIPv4(host)
	if len(ips) == 0 {
		return nil
	}
	var cidrs []string
	seen := make(map[string]struct{})
	for _, ip := range ips {
		ip4 := ip.To4()
		if ip4 == nil {
			continue
		}
		cidr := ip4.String() + "/32"
		if _, ok := seen[cidr]; ok {
			continue
		}
		seen[cidr] = struct{}{}
		cidrs = append(cidrs, cidr)
	}
	return cidrs
}

func tunRouteExcludeAddresses(proxyHost string) []string {
	exclude := make([]string, 0, len(tunExcludedCIDRs)+len(bootstrapExcludeCIDRs)+4)
	exclude = append(exclude, tunExcludedCIDRs...)
	exclude = append(exclude, bootstrapExcludeCIDRs...)
	exclude = append(exclude, resolveHostIPv4CIDRs(proxyHost)...)
	return exclude
}

// proxyServerAddress returns a dial target for the VMess outbound. When the
// ws-proxy host resolves to IPv4 at config build time, use that address so the
// proxy dials the real edge IP directly. With system DNS pointed at the TUN,
// hostname resolution via the OS loops through FakeIP and breaks the tunnel.
func buildTunRouteConfig(routeRules []map[string]any, localSocksPort int) map[string]any {
	resolver := "remote"
	if localSocksPort > 0 {
		// Required by sing-box 1.12+; user DNS still uses fakeip via DNS rules.
		resolver = "bootstrap"
	}
	return map[string]any{
		"rules":                   routeRules,
		"default_domain_resolver": resolver,
		"auto_detect_interface":   true,
		"final":                   "proxy",
	}
}

func remoteDNSDetour(localSocksPort int) string {
	if localSocksPort > 0 {
		return "proxy"
	}
	return "direct"
}

func buildTunDNSConfig(proxyHost string, localSocksPort int) map[string]any {
	useSidecarDNS := localSocksPort > 0
	servers := []map[string]any{
		{
			"type": "local",
			"tag":  "local",
		},
		{
			"type":        "udp",
			"tag":         "bootstrap",
			"server":      bootstrapDNSAddr,
			"server_port": 53,
			"detour":      "direct",
		},
		{
			"type":        "fakeip",
			"tag":         "fakeip",
			"inet4_range": fakeIPRange,
		},
	}
	rules := []map[string]any{
		{
			"query_type": []string{"PTR"},
			"domain_suffix": []string{
				".in-addr.arpa",
				".ip6.arpa",
			},
			"action": "reject",
		},
		{
			"query_type": []string{"A", "AAAA"},
			"action":     "route",
			"server":     "fakeip",
		},
	}
	if useSidecarDNS {
		// xray resolves hostnames via SOCKS; fakeip answers instantly.
		rules = append(rules, map[string]any{
			"action": "route",
			"server": "fakeip",
		})
	} else {
		servers = append(servers,
			map[string]any{
				"type":        "udp",
				"tag":         "bootstrap",
				"server":      bootstrapDNSAddr,
				"server_port": 53,
				"detour":      "direct",
			},
			remoteDNSServer(localSocksPort),
		)
		rules = append([]map[string]any{{
			"domain": []string{proxyHost},
			"action": "route",
			"server": "bootstrap",
		}}, rules...)
		rules = append(rules, map[string]any{
			"action": "route",
			"server": "remote",
		})
	}
	return map[string]any{
		"servers":  servers,
		"rules":    rules,
		"strategy": "ipv4_only",
	}
}

func remoteDNSServer(localSocksPort int) map[string]any {
	server := map[string]any{
		"type":        "udp",
		"tag":         "remote",
		"server":      remoteDNSAddr,
		"server_port": 53,
		"detour":      remoteDNSDetour(localSocksPort),
	}
	return server
}

func proxyServerAddress(host string, proxyIPs []string) string {
	if len(proxyIPs) == 0 {
		return host
	}
	return strings.TrimSuffix(proxyIPs[0], "/32")
}

func BuildSingBoxTunConfig(vmess *VMessParams, opts *BuildConfigOptions) ([]byte, error) {
	port := 443
	if vmess.Port != "" {
		if p, err := strconv.Atoi(vmess.Port); err == nil && p > 0 {
			port = p
		}
	}
	alterID := 0
	if vmess.AlterID != "" {
		if a, err := strconv.Atoi(vmess.AlterID); err == nil {
			alterID = a
		}
	}
	tlsEnabled := vmess.TLS == "tls"
	proxyIPs := resolveHostIPv4CIDRs(vmess.Host)

	bindIface := ""
	localSocksPort := 0
	if opts != nil {
		bindIface = opts.BindInterface
		localSocksPort = opts.LocalSocksPort
	}

	routeRules := []map[string]any{
		{"action": "sniff"},
	}
	if localSocksPort == 0 {
		routeRules = append(routeRules, map[string]any{
			"action":   "resolve",
			"server":   "remote",
			"strategy": "ipv4_only",
		})
	}
	routeRules = append(routeRules,
		map[string]any{
			"type": "logical",
			"mode": "or",
			"rules": []map[string]any{
				{"protocol": "dns"},
				{"port": 53},
			},
			"action": "hijack-dns",
		},
		map[string]any{
			"action":   "route",
			"domain":   []string{vmess.Host},
			"outbound": "direct",
		},
	)
	if len(proxyIPs) > 0 {
		routeRules = append(routeRules, map[string]any{
			"action":   "route",
			"ip_cidr":  proxyIPs,
			"outbound": "direct",
		})
	}
	routeRules = append(routeRules, map[string]any{
		"action":   "route",
		"ip_cidr":  lanBypassCIDRs,
		"outbound": "direct",
	})

	var proxyOutbound map[string]any
	if localSocksPort > 0 {
		proxyOutbound = map[string]any{
			"type":        "socks",
			"tag":         "proxy",
			"server":      "127.0.0.1",
			"server_port": localSocksPort,
			"version":     "5",
			"udp_over_tcp": map[string]any{
				"enabled": true,
				"version": 2,
			},
		}
	} else {
		proxyServer := proxyServerAddress(vmess.Host, proxyIPs)
		proxyOutbound = map[string]any{
			"type":        "vmess",
			"tag":         "proxy",
			"server":      proxyServer,
			"server_port": port,
			"uuid":        vmess.UUID,
			"security":    "auto",
			"alter_id":    alterID,
			"transport": map[string]any{
				"type": "ws",
				"path": vmess.Path,
				"headers": map[string]any{
					"Host": vmess.Host,
				},
			},
			"tls": map[string]any{
				"enabled":     tlsEnabled,
				"server_name": vmess.Host,
				"utls": map[string]any{
					// uTLS Chrome fingerprint makes Cloudflare return HTTP 404 on the
					// ws-proxy WebSocket upgrade; standard TLS matches xray and works.
					"enabled": false,
				},
			},
		}
		if proxyServer == vmess.Host {
			proxyOutbound["domain_resolver"] = "bootstrap"
		}
	}
	directOutbound := map[string]any{
		"type": "direct",
		"tag":  "direct",
	}
	if bindIface != "" {
		// bind_interface on the proxy outbound breaks sing-box native VMess under
		// TUN; the xray sidecar path dials ws-proxy from userspace without TUN.
		if localSocksPort == 0 {
			proxyOutbound["bind_interface"] = bindIface
		}
		directOutbound["bind_interface"] = bindIface
	}
	outbounds := []map[string]any{proxyOutbound, directOutbound}

	cfg := map[string]any{
		"log": map[string]any{
			"level":  "info",
			"output": singBoxLogPath(),
		},
		"dns": buildTunDNSConfig(vmess.Host, localSocksPort),
		"inbounds": []map[string]any{
			{
				"type":                   "tun",
				"tag":                    "tun-in",
				"address":                []string{"172.19.0.1/30"},
				"mtu":                    1280,
				"auto_route":             true,
				"strict_route":           true,
				"stack":                  "system",
				"route_exclude_address":  tunRouteExcludeAddresses(vmess.Host),
			},
		},
		"outbounds": outbounds,
		"route": buildTunRouteConfig(routeRules, localSocksPort),
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal sing-box config: %w", err)
	}
	return data, nil
}
