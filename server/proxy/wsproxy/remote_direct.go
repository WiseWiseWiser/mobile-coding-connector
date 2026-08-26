package wsproxy

import (
	"fmt"
	"strconv"
	"strings"
)

// RemoteDirectPattern is a parsed --remote-direct / API pattern.
// Grammar matches client --also-proxy: host, host:port, *.zone, *.zone:port.
type RemoteDirectPattern struct {
	Raw      string
	Wildcard bool
	Host     string // exact host, or suffix ".zone" for wildcard
	Port     int    // 0 = all ports
}

// ParseRemoteDirectPatterns validates and dedupes remote-direct patterns.
func ParseRemoteDirectPatterns(raw []string) ([]RemoteDirectPattern, error) {
	seen := make(map[string]struct{})
	var out []RemoteDirectPattern
	for _, item := range raw {
		p, err := parseRemoteDirectPattern(item)
		if err != nil {
			return nil, fmt.Errorf("invalid remote_direct pattern %q: %w", item, err)
		}
		if _, ok := seen[p.Raw]; ok {
			continue
		}
		seen[p.Raw] = struct{}{}
		out = append(out, p)
	}
	return out, nil
}

func parseRemoteDirectPattern(raw string) (RemoteDirectPattern, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return RemoteDirectPattern{}, fmt.Errorf("empty pattern")
	}

	hostPart := s
	port := 0
	if i := strings.LastIndex(s, ":"); i >= 0 {
		hostPart = s[:i]
		portStr := s[i+1:]
		if hostPart == "" {
			return RemoteDirectPattern{}, fmt.Errorf("port-only patterns are not supported (use host:port or *.zone:port)")
		}
		if portStr == "" {
			return RemoteDirectPattern{}, fmt.Errorf("missing port after ':'")
		}
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 1 || p > 65535 {
			return RemoteDirectPattern{}, fmt.Errorf("invalid port %q", portStr)
		}
		port = p
	}

	wildcard := false
	hostVal := hostPart
	if strings.HasPrefix(hostPart, "*.") {
		suffix := strings.TrimPrefix(hostPart, "*")
		zone := strings.TrimPrefix(suffix, ".")
		if zone == "" || strings.Contains(zone, "*") {
			return RemoteDirectPattern{}, fmt.Errorf("invalid wildcard zone")
		}
		wildcard = true
		hostVal = suffix // ".zone"
	} else if strings.Contains(hostPart, "*") {
		return RemoteDirectPattern{}, fmt.Errorf("only *.zone wildcard patterns are supported")
	} else if hostPart == "*" {
		return RemoteDirectPattern{}, fmt.Errorf("only *.zone wildcard patterns are supported")
	}

	return RemoteDirectPattern{
		Raw:      raw,
		Wildcard: wildcard,
		Host:     hostVal,
		Port:     port,
	}, nil
}

func remoteDirectRaws(patterns []RemoteDirectPattern) []string {
	out := make([]string, len(patterns))
	for i, p := range patterns {
		out[i] = p.Raw
	}
	return out
}
