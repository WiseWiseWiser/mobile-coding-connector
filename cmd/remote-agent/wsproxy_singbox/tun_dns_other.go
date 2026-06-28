//go:build !darwin

package wsproxy_singbox

func restoreStuckTunDNS() error {
	return nil
}

func configurePlatformTunDNS() (restore func(), err error) {
	return func() {}, nil
}