package wsproxy_singbox

import (
	"os"
	"strings"
	"testing"
)

func TestSingBoxProcessEnvStripsProxyVars(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8210")
	t.Setenv("ALL_PROXY", "socks5h://127.0.0.1:1080")
	t.Setenv("KEEP_ME", "yes")

	env := singBoxProcessEnv()
	joined := strings.Join(env, "\n")
	for _, key := range []string{"HTTP_PROXY", "ALL_PROXY", "http_proxy", "all_proxy"} {
		if strings.Contains(joined, key+"=") {
			t.Fatalf("proxy var %s not stripped from env:\n%s", key, joined)
		}
	}
	if !strings.Contains(joined, "KEEP_ME=yes") {
		t.Fatalf("non-proxy env removed:\n%s", joined)
	}
}

func TestHasProxyEnv(t *testing.T) {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(key, "")
	}
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8210")
	if !hasProxyEnv() {
		t.Fatal("expected hasProxyEnv true")
	}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(key, "")
	}
	if hasProxyEnv() {
		t.Fatal("expected hasProxyEnv false when proxy vars are empty")
	}
	_ = os.Getenv
}