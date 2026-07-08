//go:build unix

package run

import (
	"os"
	"testing"
)

func TestIsManagedServerChild(t *testing.T) {
	orig := os.Args
	t.Cleanup(func() { os.Args = orig })

	os.Args = []string{"ai-critic-server", "--port", "23712"}
	if !isManagedServerChild() {
		t.Fatal("keep-alive child with --port should be managed")
	}

	os.Args = []string{"ai-critic-server", "keep-alive"}
	if isManagedServerChild() {
		t.Fatal("keep-alive subcommand is not a managed server child")
	}

	os.Args = []string{"ai-critic-server"}
	if isManagedServerChild() {
		t.Fatal("bare server without --port is not a managed child")
	}
}