package wsproxy_singbox

func buildTunConfigOptions(localSocksPort int) *BuildConfigOptions {
	opts := &BuildConfigOptions{LocalSocksPort: localSocksPort}
	if iface := defaultOutboundBindInterface(); iface != "" {
		opts.BindInterface = iface
	}
	return opts
}