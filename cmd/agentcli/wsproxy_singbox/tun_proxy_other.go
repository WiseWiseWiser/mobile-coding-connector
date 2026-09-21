//go:build !darwin

package wsproxy_singbox

func defaultOutboundBindInterface() string {
	return ""
}

func systemProxyEnabled() bool {
	return false
}

func disableSystemProxiesForTun() (restore func(), err error) {
	return func() {}, nil
}

func getServiceProxyState(service string) (serviceProxyState, error) {
	return serviceProxyState{}, nil
}
