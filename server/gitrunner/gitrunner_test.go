package gitrunner

import (
	"strings"
	"testing"
)

func TestValidateSSHConfigRequiresNCWhenProxyEnabled(t *testing.T) {
	prev := isToolAvailable
	isToolAvailable = func(name string) bool { return false }
	defer func() { isToolAvailable = prev }()

	err := validateSSHConfig(&SSHKeyConfig{
		ProxyURL: "http://proxy.example.com:3128",
	})
	if err == nil {
		t.Fatalf("validateSSHConfig() error = nil, want missing nc error")
	}
	if !strings.Contains(err.Error(), "remote-agent exec apt install netcat") {
		t.Fatalf("validateSSHConfig() = %q, want install hint", err.Error())
	}
}

func TestValidateSSHConfigSkipsNCWithoutProxy(t *testing.T) {
	prev := isToolAvailable
	isToolAvailable = func(name string) bool { return false }
	defer func() { isToolAvailable = prev }()

	if err := validateSSHConfig(&SSHKeyConfig{}); err != nil {
		t.Fatalf("validateSSHConfig() error = %v, want nil", err)
	}
}

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

func TestBuildPreservesCustomAskPass(t *testing.T) {
	cmd := NewCommand("fetch").WithEnv("GIT_ASKPASS", "/tmp/git-askpass").Build()

	got := envValues(cmd.Env, "GIT_ASKPASS")
	if len(got) != 1 || got[0] != "/tmp/git-askpass" {
		t.Fatalf("GIT_ASKPASS values = %v, want only custom helper", got)
	}
}

func envValues(env []string, key string) []string {
	prefix := key + "="
	var values []string
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			values = append(values, strings.TrimPrefix(entry, prefix))
		}
	}
	return values
}
