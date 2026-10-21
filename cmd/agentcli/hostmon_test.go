package agentcli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestHostMonHelpAndQuery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := RunWithWriters(RemoteProfile(), []string{"hostmon", "--help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Subcommands:") || !strings.Contains(stdout.String(), "top [--lines N]") {
		t.Fatalf("help = %q", stdout.String())
	}

	sample := client.HostMonSample{
		T:     "2026-10-21T06:40:00+08:00",
		Load1: 6.2,
	}
	sample.CPU.User = 12
	sample.CPU.IOWait = 71
	sample.Disk.Dev = "vda"
	sample.Disk.RIops = 1804
	sample.Disk.WIops = 2
	sample.Disk.RBps = 4.2 * 1024 * 1024
	sample.Top = []client.HostMonProc{{PID: 1842, Comm: "cloudflared", ReadBps: 6.1 * 1024 * 1024}}

	status := client.HostMonStatus{
		Running: false, AutoStart: false, IntervalSec: 15,
		LogPath: "/root/.ai-critic/hostmon/samples.log", LogBytes: 12 * 1024,
		Latest: &sample,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/remote-agent/hostmon/status":
			_ = json.NewEncoder(w).Encode(status)
		case r.URL.Path == "/api/remote-agent/hostmon/samples":
			_ = json.NewEncoder(w).Encode(map[string]any{"samples": []client.HostMonSample{sample}})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/remote-agent/hostmon/"):
			status.Running = r.URL.Path == "/api/remote-agent/hostmon/start"
			status.AutoStart = r.URL.Path == "/api/remote-agent/hostmon/enable"
			_ = json.NewEncoder(w).Encode(status)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	stdout.Reset()
	stderr.Reset()
	args := []string{"--server", srv.URL, "--token", "t", "hostmon", "top"}
	if err := RunWithWriters(RemoteProfile(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "warning: sampler is stopped; showing the last persisted sample") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "load 6.2") || !strings.Contains(stdout.String(), "1842") || !strings.Contains(stdout.String(), "cloudflared") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := RunWithWriters(RemoteProfile(), []string{"--server", srv.URL, "--token", "t", "hostmon", "log", "--lines", "1"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "top=1842 cloudflared") {
		t.Fatalf("log stdout = %q", stdout.String())
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "samples") {
			_, _ = io.WriteString(w, `{"samples":[]}`)
			return
		}
		_, _ = io.WriteString(w, `{"running":false,"autoStart":false,"intervalSec":15,"logPath":"/tmp/samples.log","logBytes":0}`)
	}))
	t.Cleanup(empty.Close)
	err := RunWithWriters(RemoteProfile(), []string{"--server", empty.URL, "--token", "t", "hostmon", "top"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "hostmon has no samples yet") {
		t.Fatalf("err = %v", err)
	}
}
