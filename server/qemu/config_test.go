package qemu

import (
	"path/filepath"
	"testing"

	shared "github.com/xhd2015/dot-pkgs/go-pkgs/qemu"
)

func TestEnabledDefaultFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "qemu.json")
	prev := getConfigFile()
	SetConfigFile(path)
	t.Cleanup(func() { SetConfigFile(prev) })

	if Enabled() {
		t.Fatal("missing qemu.json must be disabled")
	}
}

func TestEnabledTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "qemu.json")
	if err := shared.SaveFileConfig(path, shared.FileConfig{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	prev := getConfigFile()
	SetConfigFile(path)
	t.Cleanup(func() { SetConfigFile(prev) })

	if !Enabled() {
		t.Fatal("expected enabled")
	}
	cfg, err := LoadConfig()
	if err != nil || !cfg.Enabled {
		t.Fatalf("LoadConfig: %v %#v", err, cfg)
	}
}

func TestGuestOriginForPort(t *testing.T) {
	if got := GuestOriginForPort(23712); got != "http://10.0.2.2:23712" {
		t.Fatalf("got %q", got)
	}
}
