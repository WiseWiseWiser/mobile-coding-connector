package cloudflareproxy

import (
	"context"
	"net"
	"os"
	"testing"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func shortBase(t *testing.T) string {
	t.Helper()
	base, err := os.MkdirTemp("/tmp", "cfp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	return base
}

func TestControlEnableNowStarts(t *testing.T) {
	base := shortBase(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := Host(ctx, HostOptions{BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	resp, err := ControlClient(SocketPath(base), OpRequest{Op: "enable", Now: true, Port: freePort(t)})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("enable: %v", resp.Error)
	}
	if resp.KV["status"] != "running" {
		t.Fatalf("status: %v", resp.KV)
	}
	if resp.KV["autoStart"] != "true" {
		t.Fatalf("autoStart: %v", resp.KV)
	}
	cfg, err := LoadConfig(ConfigPath(base))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoStart {
		t.Fatal("config autoStart")
	}
	if cfg.GeneratedToken == "" {
		t.Fatal("expected generatedToken")
	}
}

func TestControlStopNotRunningWarns(t *testing.T) {
	base := shortBase(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := Host(ctx, HostOptions{BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	resp, err := ControlClient(SocketPath(base), OpRequest{Op: "stop"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("stop: %v", resp.Error)
	}
	if resp.Output != "warning: cloudflare-proxy is not running" {
		t.Fatalf("output %q", resp.Output)
	}
}

func TestBootAutoStart(t *testing.T) {
	base := shortBase(t)
	cfg := Config{AutoStart: true, Port: freePort(t), Token: "fixed"}
	if err := SaveConfig(ConfigPath(base), cfg); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := Host(ctx, HostOptions{BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	resp, err := ControlClient(SocketPath(base), OpRequest{Op: "status"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.KV["status"] != "running" {
		t.Fatalf("boot status: %v", resp.KV)
	}
	if resp.KV["auth"] != "token" {
		t.Fatalf("auth: %v", resp.KV)
	}
}
