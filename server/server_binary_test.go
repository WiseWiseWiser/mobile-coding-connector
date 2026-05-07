package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNextBinaryTargetUsesHighestExistingVersion(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"ai-critic-server-v1",
		"ai-critic-server-v2",
		"ai-critic-server-v10",
		"ai-critic-server-v11",
		"ai-critic-server-v12",
		"ai-critic-server-v13",
		"ai-critic-server-v15",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	target, err := nextBinaryTargetForPath(filepath.Join(dir, "ai-critic-server-v13"))
	if err != nil {
		t.Fatalf("nextBinaryTargetForPath() error = %v", err)
	}
	if target.Version != 16 {
		t.Fatalf("target.Version = %d, want 16", target.Version)
	}
	if target.BinaryName != "ai-critic-server-v16" {
		t.Fatalf("target.BinaryName = %q, want ai-critic-server-v16", target.BinaryName)
	}
}

func TestNextBinaryTargetStartsAtV1ForUnversionedBinary(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "ai-critic-server")
	if err := os.WriteFile(current, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	target, err := nextBinaryTargetForPath(current)
	if err != nil {
		t.Fatalf("nextBinaryTargetForPath() error = %v", err)
	}
	if target.BinaryName != "ai-critic-server-v1" {
		t.Fatalf("target.BinaryName = %q, want ai-critic-server-v1", target.BinaryName)
	}
}
