package wsproxy_singbox

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AlsoProxyPattern is a parsed --also-proxy entry (http-only TCP exceptions).
type AlsoProxyPattern struct {
	Raw      string
	Wildcard bool
	Host     string // exact host, or suffix ".zone" for wildcard (same as DomainPattern.Value)
	Port     int    // 0 = all ports
}

// ParseAlsoProxyPatterns validates and dedupes --also-proxy values.
// Supported: host, host:port, *.zone, *.zone:port. Port-only (:port) is rejected.
func ParseAlsoProxyPatterns(raw []string) ([]AlsoProxyPattern, error) {
	seen := make(map[string]struct{})
	var out []AlsoProxyPattern
	for _, item := range raw {
		pattern, err := parseAlsoProxyPattern(item)
		if err != nil {
			return nil, fmt.Errorf("invalid --also-proxy %q: %w", item, err)
		}
		key := pattern.Raw
		if _, ok := seen[key]; ok {
			fmt.Fprintf(os.Stderr, "warning: duplicate --also-proxy %q (ignored)\n", key)
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pattern)
	}
	return out, nil
}

func parseAlsoProxyPattern(raw string) (AlsoProxyPattern, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return AlsoProxyPattern{}, fmt.Errorf("empty pattern")
	}

	hostPart := s
	port := 0
	if i := strings.LastIndex(s, ":"); i >= 0 {
		hostPart = s[:i]
		portStr := s[i+1:]
		if hostPart == "" {
			return AlsoProxyPattern{}, fmt.Errorf("port-only patterns are not supported (use host:port or *.zone:port)")
		}
		if portStr == "" {
			return AlsoProxyPattern{}, fmt.Errorf("missing port after ':'")
		}
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 1 || p > 65535 {
			return AlsoProxyPattern{}, fmt.Errorf("invalid port %q", portStr)
		}
		port = p
	}

	dom, err := parseDomainPattern(hostPart)
	if err != nil {
		return AlsoProxyPattern{}, err
	}
	return AlsoProxyPattern{
		Raw:      raw,
		Wildcard: dom.Wildcard,
		Host:     dom.Value,
		Port:     port,
	}, nil
}

func appendAlsoProxyRouteRules(rules []map[string]any, patterns []AlsoProxyPattern, outbound string) []map[string]any {
	for _, p := range patterns {
		rules = append(rules, alsoProxyRouteRule(p, outbound))
	}
	return rules
}

func alsoProxyRouteRule(pattern AlsoProxyPattern, outbound string) map[string]any {
	rule := map[string]any{
		"action":   "route",
		"outbound": outbound,
	}
	if pattern.Wildcard {
		rule["domain_suffix"] = []string{pattern.Host}
	} else {
		rule["domain"] = []string{pattern.Host}
	}
	if pattern.Port > 0 {
		rule["port"] = pattern.Port
	}
	return rule
}
