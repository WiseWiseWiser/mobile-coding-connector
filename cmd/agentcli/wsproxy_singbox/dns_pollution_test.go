package wsproxy_singbox

import (
	"testing"
)

func TestIsLikelyGoogleIPv4(t *testing.T) {
	if !isLikelyGoogleIPv4("142.250.1.2") {
		t.Fatal("expected Google range match")
	}
	if isLikelyGoogleIPv4("31.13.92.37") {
		t.Fatal("facebook-ish IP should not match Google")
	}
}

func TestCheckDNSPollutionDetectsMismatch(t *testing.T) {
	oldSystem := lookupSystemHostIPv4
	oldTrusted := lookupTrustedHostIPv4
	defer func() {
		lookupSystemHostIPv4 = oldSystem
		lookupTrustedHostIPv4 = oldTrusted
	}()

	lookupSystemHostIPv4 = func(host string) (string, error) {
		return "31.13.92.37", nil
	}
	lookupTrustedHostIPv4 = func(host string) (string, error) {
		return "142.250.198.142", nil
	}

	check := CheckDNSPollution()
	if !check.Polluted {
		t.Fatalf("check = %#v, want polluted", check)
	}
}

func TestCheckDNSPollutionClean(t *testing.T) {
	oldSystem := lookupSystemHostIPv4
	oldTrusted := lookupTrustedHostIPv4
	defer func() {
		lookupSystemHostIPv4 = oldSystem
		lookupTrustedHostIPv4 = oldTrusted
	}()

	lookupSystemHostIPv4 = func(host string) (string, error) {
		return "142.250.198.142", nil
	}
	lookupTrustedHostIPv4 = func(host string) (string, error) {
		return "142.250.198.142", nil
	}

	check := CheckDNSPollution()
	if check.Polluted {
		t.Fatalf("check = %#v, want clean", check)
	}
}