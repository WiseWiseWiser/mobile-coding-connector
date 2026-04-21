package git

import (
	"strings"
	"testing"
)

func TestRedactURLSecretsMasksPasswordButKeepsHost(t *testing.T) {
	got := redactURLSecrets("http://alice:secret@example.com:3128")
	if got != "http://alice:%3Credacted%3E@example.com:3128" {
		t.Fatalf("redactURLSecrets() = %q", got)
	}
}

func TestRedactSSHCommandSecretsMasksIdentityFileAndProxyPassword(t *testing.T) {
	cmd := `"ssh" "-i" "/tmp/op-key-123" "-o" "ProxyCommand=curl http://alice:secret@example.com:3128"`
	got := redactSSHCommandSecrets(cmd)

	for _, want := range []string{
		`"<redacted-private-key-path>"`,
		`alice:%3Credacted%3E@example.com:3128`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("redactSSHCommandSecrets() missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "/tmp/op-key-123") || strings.Contains(got, "secret") {
		t.Fatalf("redactSSHCommandSecrets() leaked secret data in %q", got)
	}
}
