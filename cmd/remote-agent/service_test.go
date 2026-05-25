package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

func TestMatchServiceTargetPrefersID(t *testing.T) {
	services := []client.ServiceStatus{
		{ID: "svc-1", Name: "web"},
		{ID: "web", Name: "other"},
	}

	got, err := matchServiceTarget(services, "web")
	if err != nil {
		t.Fatalf("matchServiceTarget() error = %v", err)
	}
	if got.ID != "web" {
		t.Fatalf("matchServiceTarget() ID = %q, want %q", got.ID, "web")
	}
}

func TestMatchServiceTargetRejectsAmbiguousName(t *testing.T) {
	services := []client.ServiceStatus{
		{ID: "svc-1", Name: "web"},
		{ID: "svc-2", Name: "web"},
	}

	_, err := matchServiceTarget(services, "web")
	if err == nil {
		t.Fatalf("matchServiceTarget() error = nil, want ambiguity error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("matchServiceTarget() error = %q, want ambiguity message", err.Error())
	}
}

func TestRunServiceUpgradeUploadsBeforeRemoteUpgrade(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	localBinary := filepath.Join(t.TempDir(), "server-linux-amd64")
	if err := os.WriteFile(localBinary, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var (
		events     []string
		uploadPath string
		upgradeReq client.ServiceUpgradeRequest
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/services":
			events = append(events, "list")
			writeJSON(w, []client.ServiceStatus{{ID: "svc-1", Name: "web", Status: "running"}})
		case "/api/files/home":
			events = append(events, "home")
			writeJSON(w, map[string]string{"home": "/home/agent", "cwd": "/home/agent"})
		case "/api/files/upload/init":
			events = append(events, "upload-init")
			var req struct {
				Path      string `json:"path"`
				ChmodExec bool   `json:"chmod_exec"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode upload init: %v", err)
			}
			uploadPath = req.Path
			if !req.ChmodExec {
				t.Fatalf("upload init chmod_exec = false, want true")
			}
			writeJSON(w, map[string]string{"upload_id": "upload-1"})
		case "/api/files/upload/chunk":
			events = append(events, "upload-chunk")
			writeJSON(w, map[string]any{"status": "ok"})
		case "/api/files/upload/complete":
			events = append(events, "upload-complete")
			writeJSON(w, client.UploadResult{Status: "ok", Path: uploadPath, Size: 6})
		case "/api/services/upgrade":
			events = append(events, "upgrade")
			var req client.ServiceUpgradeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode upgrade request: %v", err)
			}
			upgradeReq = req
			writeJSON(w, client.ServiceUpgradeResult{
				Status:     "ok",
				TmpPath:    req.TmpPath,
				TargetPath: "/home/agent/bin/server",
				Service:    &client.ServiceStatus{ID: "svc-1", Name: "web", Status: "running", PID: 123},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	resolve := func() (*client.Client, error) {
		return client.New(server.URL, ""), nil
	}
	if err := runServiceUpgrade(resolve, []string{"web", localBinary, "--target", "~/bin/server"}); err != nil {
		t.Fatalf("runServiceUpgrade() error = %v", err)
	}

	wantEvents := []string{"list", "upload-init", "upload-chunk", "upload-complete", "upgrade"}
	if strings.Join(events, ",") != strings.Join(wantEvents, ",") {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
	if uploadPath == "" || !strings.HasPrefix(uploadPath, "/tmp/remote-agent-upgrade-") {
		t.Fatalf("upload path = %q, want /tmp/remote-agent-upgrade-*", uploadPath)
	}
	if upgradeReq.ID != "svc-1" {
		t.Fatalf("upgrade id = %q, want svc-1", upgradeReq.ID)
	}
	if upgradeReq.TmpPath != uploadPath {
		t.Fatalf("upgrade tmp path = %q, want upload path %q", upgradeReq.TmpPath, uploadPath)
	}
	if upgradeReq.LocalBase != "server-linux-amd64" {
		t.Fatalf("upgrade local base = %q, want server-linux-amd64", upgradeReq.LocalBase)
	}
	if upgradeReq.Target != "~/bin/server" {
		t.Fatalf("upgrade target = %q, want ~/bin/server", upgradeReq.Target)
	}
}
