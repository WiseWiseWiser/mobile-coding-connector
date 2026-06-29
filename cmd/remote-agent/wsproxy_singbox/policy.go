package wsproxy_singbox

import (
	"fmt"
	"os"
	"strings"
)

// PolicyMode controls which HTTP/HTTPS flows use the ws-proxy path.
type PolicyMode int

const (
	PolicyBlacklist PolicyMode = iota
	PolicyWhitelist
)

func (m PolicyMode) String() string {
	switch m {
	case PolicyWhitelist:
		return "whitelist"
	default:
		return "blacklist"
	}
}

// DomainPattern is a parsed --include/--exclude entry.
type DomainPattern struct {
	Raw      string
	Wildcard bool
	Value    string // exact host, or suffix ".zone" for wildcard
}

// DomainPolicy is the resolved routing policy for vpn / vpn --http-only.
type DomainPolicy struct {
	Mode    PolicyMode
	Include []DomainPattern
	Exclude []DomainPattern
}

// PolicyInput is raw CLI input before validation.
type PolicyInput struct {
	Whitelist bool
	Blacklist bool
	Include   []string
	Exclude   []string
}

// ParseDomainPolicy validates flags and returns a routing policy.
func ParseDomainPolicy(in PolicyInput) (*DomainPolicy, error) {
	if in.Whitelist && in.Blacklist {
		return nil, fmt.Errorf("--whitelist and --blacklist are mutually exclusive")
	}

	include, err := parseAndDedupePatterns(in.Include, "include")
	if err != nil {
		return nil, err
	}
	exclude, err := parseAndDedupePatterns(in.Exclude, "exclude")
	if err != nil {
		return nil, err
	}

	mode, err := resolvePolicyMode(in.Whitelist, in.Blacklist, include, exclude)
	if err != nil {
		return nil, err
	}

	if err := validatePolicySubdomains(mode, include, exclude); err != nil {
		return nil, err
	}

	return &DomainPolicy{
		Mode:    mode,
		Include: include,
		Exclude: exclude,
	}, nil
}

func resolvePolicyMode(whitelist, blacklist bool, include, exclude []DomainPattern) (PolicyMode, error) {
	switch {
	case whitelist:
		return PolicyWhitelist, nil
	case blacklist:
		return PolicyBlacklist, nil
	case len(include) > 0 && len(exclude) > 0:
		return 0, fmt.Errorf("--whitelist or --blacklist is required when both --include and --exclude are set")
	case len(include) > 0:
		return PolicyWhitelist, nil
	default:
		return PolicyBlacklist, nil
	}
}

func parseAndDedupePatterns(raw []string, label string) ([]DomainPattern, error) {
	seen := make(map[string]struct{})
	var out []DomainPattern
	for _, item := range raw {
		pattern, err := parseDomainPattern(item)
		if err != nil {
			return nil, fmt.Errorf("invalid --%s %q: %w", label, item, err)
		}
		key := pattern.Raw
		if _, ok := seen[key]; ok {
			fmt.Fprintf(os.Stderr, "warning: duplicate domain pattern %q (ignored)\n", key)
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pattern)
	}
	return out, nil
}

func parseDomainPattern(raw string) (DomainPattern, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return DomainPattern{}, fmt.Errorf("empty pattern")
	}
	if s == "*" || strings.Contains(s, "*") && !strings.HasPrefix(s, "*.") {
		return DomainPattern{}, fmt.Errorf("only *.zone wildcard patterns are supported")
	}
	if strings.HasPrefix(s, "*.") {
		suffix := strings.TrimPrefix(s, "*")
		if len(suffix) < 2 || suffix[0] != '.' {
			return DomainPattern{}, fmt.Errorf("wildcard must be *.zone")
		}
		zone := strings.TrimPrefix(suffix, ".")
		if zone == "" || strings.Contains(zone, "*") {
			return DomainPattern{}, fmt.Errorf("invalid wildcard zone")
		}
		return DomainPattern{Raw: raw, Wildcard: true, Value: suffix}, nil
	}
	if strings.Contains(s, "*") {
		return DomainPattern{}, fmt.Errorf("only *.zone wildcard patterns are supported")
	}
	return DomainPattern{Raw: raw, Wildcard: false, Value: s}, nil
}

func validatePolicySubdomains(mode PolicyMode, include, exclude []DomainPattern) error {
	switch mode {
	case PolicyWhitelist:
		for _, child := range exclude {
			if !patternCoveredByAny(child, include) {
				return fmt.Errorf("exclude %q is not a sub-domain of any --include", child.Raw)
			}
		}
	case PolicyBlacklist:
		for _, child := range include {
			if !patternCoveredByAny(child, exclude) {
				return fmt.Errorf("include %q is not a sub-domain of any --exclude", child.Raw)
			}
		}
	}
	return nil
}

func patternCoveredByAny(child DomainPattern, parents []DomainPattern) bool {
	for _, parent := range parents {
		if patternCovers(parent, child) {
			return true
		}
	}
	return false
}

func patternCovers(parent, child DomainPattern) bool {
	for _, host := range sampleHostsForPattern(child) {
		if !hostMatchesPattern(host, parent) {
			return false
		}
	}
	return true
}

func sampleHostsForPattern(p DomainPattern) []string {
	if p.Wildcard {
		return []string{"a" + p.Value, "a.b" + p.Value}
	}
	return []string{p.Value}
}

func hostMatchesPattern(host string, p DomainPattern) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	if p.Wildcard {
		zone := strings.TrimPrefix(p.Value, ".")
		return host == zone || strings.HasSuffix(host, p.Value)
	}
	if host == p.Value {
		return true
	}
	return strings.HasSuffix(host, "."+p.Value)
}

func domainRouteRule(pattern DomainPattern, outbound string) map[string]any {
	if pattern.Wildcard {
		return map[string]any{
			"action":        "route",
			"domain_suffix": []string{pattern.Value},
			"outbound":      outbound,
		}
	}
	return map[string]any{
		"action":   "route",
		"domain":   []string{pattern.Value},
		"outbound": outbound,
	}
}