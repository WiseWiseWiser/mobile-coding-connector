package wsproxy_singbox

import (
	"os"
	"strings"
)

var proxyEnvKeys = map[string]struct{}{
	"HTTP_PROXY":  {},
	"http_proxy":  {},
	"HTTPS_PROXY": {},
	"https_proxy": {},
	"ALL_PROXY":   {},
	"all_proxy":   {},
	"NO_PROXY":    {},
	"no_proxy":    {},
}

// EnvWithoutProxy returns os.Environ() without HTTP/SOCKS proxy variables.
// Traffic through the TUN must not use shell or system HTTP proxies.
func EnvWithoutProxy() []string {
	return singBoxProcessEnv()
}

// singBoxProcessEnv returns an environment without HTTP/SOCKS proxy variables.
// sing-box must dial the ws-proxy WebSocket endpoint directly; routing that
// through a local HTTP proxy yields HTTP 404 and breaks the tunnel.
func singBoxProcessEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if _, strip := proxyEnvKeys[key]; strip {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func hasProxyEnv() bool {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}