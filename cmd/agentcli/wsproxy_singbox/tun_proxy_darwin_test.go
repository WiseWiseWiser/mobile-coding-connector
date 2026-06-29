//go:build darwin

package wsproxy_singbox

import "testing"

func TestParseProxyEndpoint(t *testing.T) {
	sample := `Enabled: Yes
Server: 127.0.0.1
Port: 1087
Authenticated Proxy Enabled: 0`
	ep := parseProxyEndpoint(sample)
	if !ep.enabled {
		t.Fatal("expected enabled")
	}
	if ep.server != "127.0.0.1" || ep.port != 1087 {
		t.Fatalf("endpoint = %#v", ep)
	}
}