package wsproxy_singbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeLogPathsUnderAgentCacheDir(t *testing.T) {
	dir, err := agentCacheDir()
	if err != nil {
		t.Fatalf("agentCacheDir: %v", err)
	}
	if !strings.HasSuffix(dir, filepath.Join("remote-agent")) {
		t.Fatalf("agentCacheDir = %q, want .../remote-agent", dir)
	}
	for _, path := range []string{xraySidecarLogPath(), singBoxLogPath()} {
		if !strings.HasPrefix(path, dir+string(os.PathSeparator)) {
			t.Fatalf("log path %q not under %q", path, dir)
		}
	}
}