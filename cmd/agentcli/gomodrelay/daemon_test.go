package gomodrelay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func httpTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func shortBase(t *testing.T) string {
	t.Helper()
	base, err := os.MkdirTemp("/tmp", "gmr-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	return base
}

func TestControlEnableNowStartsRelay(t *testing.T) {
	base := shortBase(t)
	up := httpTestServer()
	defer up.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := Host(ctx, HostOptions{BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	resp, err := ControlClient(SocketPath(base), OpRequest{Op: "enable", Now: true, Port: 29911, Upstream: up.URL, DownRecheckMS: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("enable: %v", resp.Error)
	}
	if resp.KV["status"] != "running" {
		t.Fatalf("status after enable --now: %v", resp.KV)
	}
	if resp.KV["auto_start"] != "enabled" {
		t.Fatalf("auto_start: %v", resp.KV)
	}

	// config persisted with enabled=true
	cfg, err := LoadConfig(ConfigPath(base))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.Upstream != up.URL {
		t.Fatalf("config: %+v", cfg)
	}

	// status via IPC shows running + upstream
	resp, err = ControlClient(SocketPath(base), OpRequest{Op: "status"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.KV["upstream"] != up.URL {
		t.Fatalf("status upstream: %v", resp.KV)
	}
}

func TestDaemonRebootAutoStart(t *testing.T) {
	base := shortBase(t)
	up := httpTestServer()
	defer up.Close()

	ctx1, cancel1 := context.WithCancel(context.Background())
	h1, err := Host(ctx1, HostOptions{BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ControlClient(SocketPath(base), OpRequest{Op: "enable", Now: true, Port: 29912, Upstream: up.URL, DownRecheckMS: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("enable: %v", resp.Error)
	}
	cancel1()
	_ = h1.Close()
	time.Sleep(50 * time.Millisecond)

	// simulate daemon reboot (fresh Host on same base dir)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	h2, err := Host(ctx2, HostOptions{BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer h2.Close()

	resp, err = ControlClient(SocketPath(base), OpRequest{Op: "status"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.KV["status"] != "running" {
		t.Fatalf("after daemon reboot, enabled relay should auto-start: %v", resp.KV)
	}
	if resp.KV["auto_start"] != "enabled" {
		t.Fatalf("auto_start: %v", resp.KV)
	}
}

func TestDaemonDownError(t *testing.T) {
	base := shortBase(t) // no daemon: socket never created
	_, err := ControlClient(SocketPath(base), OpRequest{Op: "status"})
	if err == nil {
		t.Fatal("expected ErrDaemonDown")
	}
	if err != ErrDaemonDown {
		t.Fatalf("want ErrDaemonDown, got %v", err)
	}
}

func TestStopAndDisable(t *testing.T) {
	base := shortBase(t)
	up := httpTestServer()
	defer up.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := Host(ctx, HostOptions{BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	if _, err := ControlClient(SocketPath(base), OpRequest{Op: "enable", Now: true, Port: 29913, Upstream: up.URL, DownRecheckMS: 100}); err != nil {
		t.Fatal(err)
	}
	resp, err := ControlClient(SocketPath(base), OpRequest{Op: "stop"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.KV["status"] != "stopped" {
		t.Fatalf("stop: %v", resp.KV)
	}
	// auto-start flag still enabled after stop (orthogonal)
	if resp.KV["auto_start"] != "enabled" {
		t.Fatalf("auto_start after stop: %v", resp.KV)
	}
	resp, err = ControlClient(SocketPath(base), OpRequest{Op: "disable", Now: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.KV["auto_start"] != "disabled" || resp.KV["status"] != "stopped" {
		t.Fatalf("disable --now: %v", resp.KV)
	}
	cfg, _ := LoadConfig(ConfigPath(base))
	if cfg.Enabled {
		t.Fatal("disable should persist enabled=false")
	}
}
