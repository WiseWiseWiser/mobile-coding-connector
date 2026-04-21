package gitrunner

import (
	"strings"
	"testing"
)

func TestBuildSSHCommandIncludesKeyAndHTTPProxy(t *testing.T) {
	cmd := buildSSHCommand(&SSHKeyConfig{
		KeyPath:  "/tmp/test key",
		ProxyURL: "http://proxy.example.com:3128",
	})

	for _, want := range []string{
		`"ssh"`,
		`"/tmp/test key"`,
		`"StrictHostKeyChecking=no"`,
		`"UserKnownHostsFile=/dev/null"`,
		`"BatchMode=yes"`,
		`ProxyCommand=`,
		`proxy.example.com:3128`,
		`\"connect\"`,
	} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("buildSSHCommand() missing %q in %q", want, cmd)
		}
	}
}

func TestBuildSSHCommandIncludesSOCKS5ProxyWithoutKey(t *testing.T) {
	cmd := buildSSHCommand(&SSHKeyConfig{
		ProxyURL: "socks5://proxy.example.com:1080",
	})
	if !strings.Contains(cmd, `proxy.example.com:1080`) || !strings.Contains(cmd, `\"5\"`) {
		t.Fatalf("buildSSHCommand() missing socks proxy config in %q", cmd)
	}
	if strings.Contains(cmd, " -i ") {
		t.Fatalf("buildSSHCommand() unexpectedly included -i in %q", cmd)
	}
}

func TestProxyCommandForURLRejectsUnsupportedSchemes(t *testing.T) {
	if got := proxyCommandForURL("https://proxy.example.com:443"); got != "" {
		t.Fatalf("proxyCommandForURL() = %q, want empty for unsupported scheme", got)
	}
}
