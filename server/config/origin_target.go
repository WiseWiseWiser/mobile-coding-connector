package config

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// OriginTarget is interpreted on the server that owns a mapping, not on its client.
// Host is a bare DNS name or IP literal, optionally with an IPv6 interface zone.
type OriginTarget struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

func (t OriginTarget) Normalize(fallbackPort int) (OriginTarget, error) {
	t.Scheme = strings.ToLower(strings.TrimSpace(t.Scheme))
	if t.Scheme == "" {
		t.Scheme = "http"
	}
	if t.Scheme != "http" && t.Scheme != "https" {
		return t, fmt.Errorf("target scheme must be http or https")
	}
	t.Host = strings.TrimSpace(t.Host)
	if t.Host == "" {
		t.Host = "127.0.0.1"
	}
	if strings.HasPrefix(t.Host, "[") && strings.HasSuffix(t.Host, "]") {
		t.Host = t.Host[1 : len(t.Host)-1]
	}
	if t.Port == 0 {
		t.Port = fallbackPort
	}
	if t.Port < 1 || t.Port > 65535 {
		return t, fmt.Errorf("target port must be between 1 and 65535")
	}
	if strings.ContainsAny(t.Host, " /?#@\\\t\r\n") {
		return t, fmt.Errorf("target host must be a bare IPv4, IPv6 or DNS host, without scheme, port or path")
	}
	if a, err := netip.ParseAddr(t.Host); err == nil {
		if err := ValidateOriginIP(a); err != nil {
			return t, err
		}
		if a.Is6() && a.IsLinkLocalUnicast() && a.Zone() == "" {
			return t, fmt.Errorf("link-local IPv6 target requires a server interface zone, e.g. %%eth0")
		}
		if a.Zone() != "" && !a.Is6() {
			return t, fmt.Errorf("only IPv6 targets support zones")
		}
		t.Host = a.String()
	} else {
		if strings.ContainsAny(t.Host, ":%[]") || len(t.Host) > 253 {
			return t, fmt.Errorf("invalid target host %q", t.Host)
		}
		t.Host = strings.ToLower(t.Host)
		for _, label := range strings.Split(strings.TrimSuffix(t.Host, "."), ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return t, fmt.Errorf("invalid DNS target host")
			}
			for _, c := range label {
				if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
					return t, fmt.Errorf("invalid DNS target host")
				}
			}
		}
	}
	return t, nil
}

// Private and loopback origins are intentional. Cloud metadata, unspecified and
// multicast destinations are not. Apply this again to every DNS result at dial time.
func ValidateOriginIP(a netip.Addr) error {
	a = a.Unmap()
	if !a.IsValid() || a.IsUnspecified() || a.IsMulticast() || a.String() == "255.255.255.255" {
		return fmt.Errorf("target IP must be a unicast address")
	}
	bare := a.WithZone("")
	for _, denied := range []string{"169.254.169.254", "169.254.170.2", "100.100.100.200", "fd00:ec2::254"} {
		if bare == netip.MustParseAddr(denied) {
			return fmt.Errorf("cloud metadata targets are not allowed")
		}
	}
	return nil
}

func (t OriginTarget) Address() string { return net.JoinHostPort(t.Host, strconv.Itoa(t.Port)) }
func (t OriginTarget) URL() string     { return (&url.URL{Scheme: t.Scheme, Host: t.Address()}).String() }
func (t OriginTarget) IsLocal() bool {
	return t.Host == "127.0.0.1" || t.Host == "localhost" || t.Host == "::1"
}
func LocalMappingID(port int) string { return fmt.Sprintf("map_local_%d", port) }
