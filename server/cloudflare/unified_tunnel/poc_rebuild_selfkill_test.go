//go:build poc

package unified_tunnel

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

// POC tests for script/tunnel-rebuild-selfkill-poc.
//   go test -tags poc -run TestPOC ./server/cloudflare/unified_tunnel/

func pocTunnelManager(t *testing.T, stopDelay time.Duration) (*UnifiedTunnelManager, func()) {
	t.Helper()

	dataDir := t.TempDir()
	oldDataDir := config.DataDir
	config.DataDir = dataDir
	t.Cleanup(func() { config.DataDir = oldDataDir })

	credPath := filepath.Join(dataDir, "tunnel-creds.json")
	if err := os.WriteFile(credPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	stopEntered := make(chan struct{}, 1)
	cleanup := SetTestProcessHooks(
		func(utm *UnifiedTunnelManager) error {
			utm.running = true
			return nil
		},
		func(utm *UnifiedTunnelManager) {
			select {
			case stopEntered <- struct{}{}:
			default:
			}
			time.Sleep(stopDelay)
			utm.running = false
			utm.cmd = nil
		},
	)

	utm := NewUnifiedTunnelManager("extension-poc")
	utm.rebuildDebounce = 20 * time.Millisecond
	utm.SetConfig(config.CloudflareTunnelConfig{
		TunnelID:        "61562913-5c12-445b-ba26-1f7e17bbf0d4",
		TunnelName:      "mcc-a0b62356-10-91-186-143-extension",
		CredentialsFile: credPath,
	})

	return utm, func() {
		cleanup()
		close(stopEntered)
	}
}

// Rebuild holds utm.mu for the entire slow stop hook; concurrent AddMapping blocks.
func TestPOCRebuildBlocksConcurrentAddMapping(t *testing.T) {
	utm, cleanup := pocTunnelManager(t, 800*time.Millisecond)
	defer cleanup()

	if err := utm.AddMapping(&IngressMapping{
		ID: "owned-port-1", Hostname: "alpha.example.com", Service: "http://localhost:4096",
	}); err != nil {
		t.Fatalf("AddMapping: %v", err)
	}
	waitForRebuildCount(t, 1, 2*time.Second)

	start := time.Now()
	done := make(chan time.Duration, 1)
	go func() {
		t0 := time.Now()
		_ = utm.AddMapping(&IngressMapping{
			ID: "owned-port-2", Hostname: "beta.example.com", Service: "http://localhost:8767",
		})
		done <- time.Since(t0)
	}()

	select {
	case blockedFor := <-done:
		if blockedFor < 400*time.Millisecond {
			t.Fatalf("concurrent AddMapping returned too fast (%v); expected block during slow rebuild stop", blockedFor)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("concurrent AddMapping blocked >3s (started %v ago)", time.Since(start))
	}
}

// Stand-in ai-critic /ping server stays healthy while rebuild holds the tunnel mutex.
func TestPOCStandInPingHealthyDuringSlowRebuild(t *testing.T) {
	utm, cleanup := pocTunnelManager(t, 900*time.Millisecond)
	defer cleanup()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "pong")
		}),
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	client := &http.Client{Timeout: 2 * time.Second}
	ping := func() (time.Duration, error) {
		t0 := time.Now()
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", port))
		if err != nil {
			return time.Since(t0), err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "pong" {
			return time.Since(t0), fmt.Errorf("bad response: %d %q", resp.StatusCode, body)
		}
		return time.Since(t0), nil
	}

	var maxLatency time.Duration
	var pingErrors int32
	stopPinger := make(chan struct{})
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopPinger:
				return
			case <-ticker.C:
				d, err := ping()
				if err != nil {
					atomic.AddInt32(&pingErrors, 1)
					continue
				}
				if d > maxLatency {
					maxLatency = d
				}
			}
		}
	}()

	if err := utm.AddMapping(&IngressMapping{
		ID: "port-4096", Hostname: "code-fast-apex-nest-23aed.xhd2015.xyz", Service: "http://localhost:4096",
	}); err != nil {
		t.Fatalf("AddMapping: %v", err)
	}
	waitForRebuildCount(t, 1, 2*time.Second)
	time.Sleep(200 * time.Millisecond)
	close(stopPinger)

	if atomic.LoadInt32(&pingErrors) > 0 {
		t.Fatalf("stand-in /ping had %d errors during rebuild; remote hang is not explained by tunnel mutex alone", pingErrors)
	}
	if maxLatency > 500*time.Millisecond {
		t.Fatalf("stand-in /ping max latency %v; expected fast responses", maxLatency)
	}
}

// pgrep pattern used by killOrphanedProcess should not match a stand-in ai-critic argv.
func TestPOCPgrepPatternSkipsStandInServer(t *testing.T) {
	cfgPath := ".ai-critic/cloudflare-tunnel-gen-extension.yml"

	argv := []string{
		"/root/ai-critic-server-linux-amd64",
		"--port", "23712",
	}
	joined := strings.Join(argv, " ")
	if strings.Contains(joined, "cloudflared") && strings.Contains(joined, cfgPath) {
		t.Fatalf("stand-in argv unexpectedly matches orphan pattern: %q", joined)
	}

	cloudflaredArgv := []string{
		"/usr/local/bin/cloudflared", "tunnel",
		"--config", cfgPath, "run", "mcc-a0b62356-10-91-186-143-extension",
	}
	joinedCF := strings.Join(cloudflaredArgv, " ")
	if !strings.Contains(joinedCF, "cloudflared") || !strings.Contains(joinedCF, cfgPath) {
		t.Fatalf("cloudflared argv should match pattern: %q", joinedCF)
	}
}