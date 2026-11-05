package wsproxy

import (
	"strings"
	"testing"
)

func TestXrayDownloadURLPerPlatform(t *testing.T) {
	cases := []struct{ goos, goarch, want string }{
		{"darwin", "arm64", "Xray-macos-arm64-v8a.zip"},
		{"darwin", "amd64", "Xray-macos-64.zip"},
		{"linux", "arm64", "Xray-linux-arm64-v8a.zip"},
		{"linux", "amd64", "Xray-linux-64.zip"},
	}
	for _, c := range cases {
		got := xrayDownloadURLFor(c.goos, c.goarch)
		if !strings.HasSuffix(got, "/"+c.want) {
			t.Fatalf("%s/%s: got %s, want suffix %s", c.goos, c.goarch, got, c.want)
		}
		if !strings.HasPrefix(got, "https://github.com/XTLS/Xray-core/releases/latest/download/") {
			t.Fatalf("%s/%s: unexpected base %s", c.goos, c.goarch, got)
		}
	}
}
