package common_opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsBinaryAvailable(t *testing.T) {
	if IsBinaryAvailable(filepath.Join(t.TempDir(), "missing-binary")) {
		t.Fatal("expected missing custom path to be unavailable")
	}

	dir := t.TempDir()
	if IsBinaryAvailable(dir) {
		t.Fatal("expected directory path to be unavailable")
	}

	file := filepath.Join(t.TempDir(), "opencode-stub")
	if err := os.WriteFile(file, []byte{}, 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsBinaryAvailable(file) {
		t.Fatal("expected executable file path to be available")
	}
}