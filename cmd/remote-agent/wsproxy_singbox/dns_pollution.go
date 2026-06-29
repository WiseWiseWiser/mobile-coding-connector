package wsproxy_singbox

import (
	"context"
	"fmt"
	"net"
	"os"
)

const dnsPollutionProbeHost = "www.google.com"

const dnsHijackRetryHint = "Retry with: remote-agent ws-proxy vpn-http-only --dns-hijack"

// DNSPollutionResult summarizes a system vs trusted DNS comparison.
type DNSPollutionResult struct {
	Host      string
	SystemIP  string
	TrustedIP string
	Polluted  bool
	Err       error
}

var (
	lookupSystemHostIPv4  = defaultLookupSystemHostIPv4
	lookupTrustedHostIPv4 = defaultLookupTrustedHostIPv4
)

func defaultLookupSystemHostIPv4(host string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hostLookupTimeout)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return "", err
	}
	ip := firstIPv4String(ips)
	if ip == "" {
		return "", fmt.Errorf("no IPv4 address for %s", host)
	}
	return ip, nil
}

func defaultLookupTrustedHostIPv4(host string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hostLookupTimeout)
	defer cancel()
	ips, err := lookupHostIPv4ViaUDP(ctx, "8.8.8.8:53", host)
	if err != nil {
		return "", err
	}
	ip := firstIPv4String(ips)
	if ip == "" {
		return "", fmt.Errorf("no IPv4 address for %s", host)
	}
	return ip, nil
}

func firstIPv4String(ips []net.IP) string {
	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
}

// CheckDNSPollution compares system DNS with 8.8.8.8 for www.google.com.
func CheckDNSPollution() DNSPollutionResult {
	result := DNSPollutionResult{Host: dnsPollutionProbeHost}
	systemIP, err := lookupSystemHostIPv4(dnsPollutionProbeHost)
	if err != nil {
		result.Err = fmt.Errorf("system DNS: %w", err)
		return result
	}
	result.SystemIP = systemIP

	trustedIP, err := lookupTrustedHostIPv4(dnsPollutionProbeHost)
	if err != nil {
		result.Err = fmt.Errorf("trusted DNS: %w", err)
		result.Polluted = !isLikelyGoogleIPv4(systemIP)
		return result
	}
	result.TrustedIP = trustedIP
	result.Polluted = systemIP != trustedIP || !isLikelyGoogleIPv4(systemIP)
	return result
}

func isLikelyGoogleIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return false
	}
	for _, cidr := range googleIPv4CIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

var googleIPv4CIDRs = []string{
	"142.250.0.0/15",
	"172.217.0.0/16",
	"216.58.192.0/19",
	"74.125.0.0/16",
}

// MaybeWarnDNSPollution prints guidance when system DNS looks polluted.
func MaybeWarnDNSPollution(dnsHijack bool) {
	check := CheckDNSPollution()
	if check.Err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not verify DNS for %s: %v\n", check.Host, check.Err)
		if !dnsHijack {
			fmt.Fprintf(os.Stderr, "         If HTTPS hangs, try %s\n", dnsHijackRetryHint)
		}
		return
	}
	if dnsHijack {
		if check.Polluted {
			fmt.Printf("DNS check: system %s → %s; trusted → %s (using --dns-hijack)\n",
				check.Host, check.SystemIP, check.TrustedIP)
		}
		return
	}
	if !check.Polluted {
		return
	}
	fmt.Fprintf(os.Stderr, "warning: system DNS may be polluted (common on phone hotspot).\n")
	fmt.Fprintf(os.Stderr, "         %s → %s (system), %s (8.8.8.8)\n", check.Host, check.SystemIP, check.TrustedIP)
	fmt.Fprintf(os.Stderr, "         HTTPS through vpn-http-only can hang without DNS hijack.\n")
	if check.TrustedIP != "" && !isLikelyGoogleIPv4(check.TrustedIP) {
		fmt.Fprintf(os.Stderr, "         Note: direct 8.8.8.8 is also polluted on this network; --dns-hijack resolves via ws-proxy.\n")
	}
	fmt.Fprintf(os.Stderr, "         %s\n", dnsHijackRetryHint)
}