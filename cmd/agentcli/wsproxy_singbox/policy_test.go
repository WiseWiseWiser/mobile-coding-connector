package wsproxy_singbox

import (
	"strings"
	"testing"
)

func TestParseDomainPolicyBlacklistDefault(t *testing.T) {
	p, err := ParseDomainPolicy(PolicyInput{})
	if err != nil {
		t.Fatalf("ParseDomainPolicy: %v", err)
	}
	if p.Mode != PolicyBlacklist {
		t.Fatalf("mode = %v, want blacklist", p.Mode)
	}
}

func TestParseDomainPolicyWhitelistInferred(t *testing.T) {
	p, err := ParseDomainPolicy(PolicyInput{Include: []string{"*.corp.com"}})
	if err != nil {
		t.Fatalf("ParseDomainPolicy: %v", err)
	}
	if p.Mode != PolicyWhitelist {
		t.Fatalf("mode = %v, want whitelist", p.Mode)
	}
}

func TestParseDomainPolicyBothListsRequireMode(t *testing.T) {
	_, err := ParseDomainPolicy(PolicyInput{
		Include: []string{"*.corp.com"},
		Exclude: []string{"cdn.corp.com"},
	})
	if err == nil || !strings.Contains(err.Error(), "--whitelist or --blacklist") {
		t.Fatalf("err = %v, want mode required", err)
	}
}

func TestParseDomainPolicyWhitelistHoleValidation(t *testing.T) {
	_, err := ParseDomainPolicy(PolicyInput{
		Whitelist: true,
		Include:   []string{"*.corp.com"},
		Exclude:   []string{"other.com"},
	})
	if err == nil || !strings.Contains(err.Error(), "not a sub-domain") {
		t.Fatalf("err = %v, want sub-domain validation", err)
	}

	p, err := ParseDomainPolicy(PolicyInput{
		Whitelist: true,
		Include:   []string{"*.corp.com"},
		Exclude:   []string{"cdn.corp.com"},
	})
	if err != nil {
		t.Fatalf("valid hole: %v", err)
	}
	if len(p.Exclude) != 1 {
		t.Fatalf("exclude = %#v", p.Exclude)
	}
}

func TestParseDomainPolicyDuplicateWarns(t *testing.T) {
	p, err := ParseDomainPolicy(PolicyInput{
		Include: []string{"*.corp.com", "*.corp.com", "corp.com"},
	})
	if err != nil {
		t.Fatalf("ParseDomainPolicy: %v", err)
	}
	if len(p.Include) != 2 {
		t.Fatalf("include = %#v, want deduped length 2", p.Include)
	}
}

func TestHostMatchesPattern(t *testing.T) {
	exact, _ := parseDomainPattern("corp.com")
	wild, _ := parseDomainPattern("*.corp.com")
	cases := []struct {
		host string
		p    DomainPattern
		want bool
	}{
		{"corp.com", exact, true},
		{"api.corp.com", exact, true},
		{"other.com", exact, false},
		{"api.corp.com", wild, true},
		{"corp.com", wild, true},
		{"other.com", wild, false},
	}
	for _, tc := range cases {
		if got := hostMatchesPattern(tc.host, tc.p); got != tc.want {
			t.Fatalf("hostMatchesPattern(%q, %q) = %v, want %v", tc.host, tc.p.Raw, got, tc.want)
		}
	}
}