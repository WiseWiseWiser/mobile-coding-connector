//go:build darwin

package wsproxy_singbox

import "testing"

func TestParseNetworkServiceForDevice(t *testing.T) {
	sample := `An asterisk (*) denotes that a network service is disabled.
(1) USB 10/100/1000 LAN
(Hardware Port: USB 10/100/1000 LAN, Device: en7)

(4) Wi-Fi
(Hardware Port: Wi-Fi, Device: en0)
`
	service, ok := parseNetworkServiceForDevice(sample, "en0")
	if !ok {
		t.Fatal("expected match for en0")
	}
	if service != "Wi-Fi" {
		t.Fatalf("service = %q, want Wi-Fi", service)
	}
}