package cloudflare

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"unicode"
)

const hostTunnelPrefix = "ai-critic-"

// HostPreferredTunnelName is ai-critic-<sanitized-hostname>.
func HostPreferredTunnelName(hostname string) string {
	h := sanitizeTunnelHost(hostname)
	if h == "" {
		h = "host"
	}
	name := hostTunnelPrefix + h
	if len(name) > 64 {
		name = name[:64]
		name = strings.TrimRight(name, "-")
	}
	return name
}

// HostPreferredTunnelNameUnique appends 6 hex for a name that collided without local creds.
func HostPreferredTunnelNameUnique(hostname string) string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return HostPreferredTunnelName(hostname) + "-x"
	}
	base := HostPreferredTunnelName(hostname)
	suffix := "-" + hex.EncodeToString(b)
	if len(base)+len(suffix) > 64 {
		base = strings.TrimRight(base[:64-len(suffix)], "-")
	}
	return base + suffix
}

// LocalHostname for tunnel naming (os.Hostname, may be empty).
func LocalHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

func sanitizeTunnelHost(hostname string) string {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if i := strings.IndexByte(hostname, '.'); i >= 0 {
		hostname = hostname[:i]
	}
	var b strings.Builder
	prevDash := false
	for _, r := range hostname {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 48 {
		s = strings.TrimRight(s[:48], "-")
	}
	return s
}
