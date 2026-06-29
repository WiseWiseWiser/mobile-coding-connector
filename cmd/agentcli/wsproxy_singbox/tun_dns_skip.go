package wsproxy_singbox

// skipPlatformTunDNS disables macOS system DNS changes during TUN setup.
var skipPlatformTunDNS bool

// SetSkipPlatformTunDNS controls whether foreground TUN setup rewrites system DNS.
func SetSkipPlatformTunDNS(skip bool) {
	skipPlatformTunDNS = skip
}

func shouldConfigurePlatformTunDNS() bool {
	return !skipPlatformTunDNS
}