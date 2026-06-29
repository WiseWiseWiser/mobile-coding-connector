package unified_tunnel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCloudflaredTunnelArgs(t *testing.T) {
	cfg := "/root/.ai-critic/cloudflare-tunnel-gen-extension.yml"
	args := []string{
		"cloudflared", "tunnel", "--config", cfg,
		"run", "mcc-co-10-91-188-48-af76-extension",
	}
	gotCfg, gotRef, ok := ParseCloudflaredTunnelArgs(args)
	if !ok {
		t.Fatal("expected ok")
	}
	if gotCfg != cfg {
		t.Fatalf("config = %q, want %q", gotCfg, cfg)
	}
	if gotRef != "mcc-co-10-91-188-48-af76-extension" {
		t.Fatalf("tunnel ref = %q", gotRef)
	}
}

func TestFindStaleTunnelConnectorsMissingConfig(t *testing.T) {
	oldList := listCloudflaredProcesses
	t.Cleanup(func() { listCloudflaredProcesses = oldList })

	canonical := filepath.Join(t.TempDir(), "cloudflare-tunnel-gen-extension.yml")
	if err := os.WriteFile(canonical, []byte("tunnel: test\n"), 0644); err != nil {
		t.Fatalf("write canonical config: %v", err)
	}

	listCloudflaredProcesses = func() ([]CloudflaredProcess, error) {
		return []CloudflaredProcess{
			{
				PID: 345229,
				Args: []string{
					"cloudflared", "tunnel",
					"--config", "/tmp/ai-critic-test-736311285/cloudflare-tunnel-gen-extension.yml",
					"run", "mcc-co-10-91-188-48-af76-extension",
				},
			},
			{
				PID: 112476,
				Args: []string{
					"cloudflared", "tunnel", "--config", canonical,
					"run", "mcc-co-10-91-188-48-af76-extension",
				},
			},
		}, nil
	}

	stale, err := FindStaleTunnelConnectors(
		"mcc-co-10-91-188-48-af76-extension",
		"7c6e51aa-dcdc-4b7c-b9ae-86ce5d4ec351",
		canonical,
		112476,
	)
	if err != nil {
		t.Fatalf("FindStaleTunnelConnectors: %v", err)
	}
	if len(stale) != 1 {
		t.Fatalf("stale = %#v, want one stale connector", stale)
	}
	if stale[0].PID != 345229 {
		t.Fatalf("stale pid = %d, want 345229", stale[0].PID)
	}
}

func TestFindStaleTunnelConnectorsIgnoresOtherTunnel(t *testing.T) {
	oldList := listCloudflaredProcesses
	t.Cleanup(func() { listCloudflaredProcesses = oldList })

	canonical := filepath.Join(t.TempDir(), "cloudflare-tunnel-gen-extension.yml")
	if err := os.WriteFile(canonical, []byte("tunnel: test\n"), 0644); err != nil {
		t.Fatalf("write canonical config: %v", err)
	}
	otherCfg := filepath.Join(t.TempDir(), "other.yml")
	if err := os.WriteFile(otherCfg, []byte("tunnel: other\n"), 0644); err != nil {
		t.Fatalf("write other config: %v", err)
	}

	listCloudflaredProcesses = func() ([]CloudflaredProcess, error) {
		return []CloudflaredProcess{
			{
				PID: 99,
				Args: []string{
					"cloudflared", "tunnel", "--config", otherCfg,
					"run", "other-tunnel",
				},
			},
		}, nil
	}

	stale, err := FindStaleTunnelConnectors("mcc-co-10-91-188-48-af76-extension", "", canonical, 0)
	if err != nil {
		t.Fatalf("FindStaleTunnelConnectors: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale = %#v, want none", stale)
	}
}