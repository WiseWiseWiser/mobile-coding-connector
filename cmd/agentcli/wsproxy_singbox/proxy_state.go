package wsproxy_singbox

// Shared proxy snapshot types used by foreground restore on every GOOS.

type proxyEndpoint struct {
	enabled bool
	server  string
	port    int
}

type serviceProxyState struct {
	web    proxyEndpoint
	secure proxyEndpoint
	socks  proxyEndpoint
}
