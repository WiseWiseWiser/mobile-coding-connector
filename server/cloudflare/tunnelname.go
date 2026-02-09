package cloudflare

import (
	"regexp"
	"strings"
)

const defaultTunnelNamePrefix = "ai-agent-"

// nonAlphanumericDash matches any character that is not alphanumeric or a dash.
var nonAlphanumericDash = regexp.MustCompile(`[^a-zA-Z0-9-]`)

// DefaultTunnelName derives a unique tunnel name from a domain string.
// It takes the first subdomain segment, replaces underscores with dashes,
// strips special characters, and prepends the "ai-agent-" prefix.
func DefaultTunnelName(domain string) string {
	parts := strings.SplitN(domain, ".", 2)
	subdomain := parts[0]

	// Replace underscores with dashes
	subdomain = strings.ReplaceAll(subdomain, "_", "-")

	// Strip non-alphanumeric/dash characters
	subdomain = nonAlphanumericDash.ReplaceAllString(subdomain, "")

	// Truncate long subdomains
	if len(subdomain) > 30 {
		subdomain = subdomain[:30]
	}

	// Trim trailing dashes
	subdomain = strings.TrimRight(subdomain, "-")

	return defaultTunnelNamePrefix + subdomain
}
