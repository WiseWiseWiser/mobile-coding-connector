package agentcli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
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

func TestRunServiceRenameSavesWithoutRestart(t *testing.T) {
	var saved client.ServiceDefinition
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/services":
			if r.Method == http.MethodGet {
				writeJSON(w, []client.ServiceStatus{{ID: "svc-1", Name: "web", Command: "run", WorkingDir: "/work"}})
				return
			}
			if r.Method != http.MethodPut || r.URL.Query().Get("restart") != "false" {
				t.Fatalf("save request = %s %s, want PUT /api/services?restart=false", r.Method, r.URL.String())
			}
			if err := json.NewDecoder(r.Body).Decode(&saved); err != nil {
				t.Fatalf("decode save request: %v", err)
			}
			writeJSON(w, client.ServiceStatus{ID: "svc-1", Name: saved.Name, Command: saved.Command, WorkingDir: saved.WorkingDir})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	resolve := func() (*client.Client, error) {
		return client.New(server.URL, ""), nil
	}
	if err := runServiceRename(resolve, []string{"web", "api"}); err != nil {
		t.Fatalf("runServiceRename() error = %v", err)
	}
	if saved.Name != "api" {
		t.Fatalf("saved name = %q, want api", saved.Name)
	}
	if saved.Command != "run" || saved.WorkingDir != "/work" {
		t.Fatalf("saved definition did not preserve command/working dir: %#v", saved)
	}
}

func TestRunServiceDisablePrintsMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/services":
			writeJSON(w, []client.ServiceStatus{{ID: "svc-1", Name: "web", Status: "running", PID: 42}})
		case "/api/services/disable":
			writeJSON(w, client.ServiceActionResponse{
				Status:  "ok",
				Message: "The server won't stop immediately unless you manually stop it",
				Service: &client.ServiceStatus{ID: "svc-1", Name: "web", Status: "running", PID: 42, Enabled: false},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	resolve := func() (*client.Client, error) {
		return client.New(server.URL, ""), nil
	}
	if err := runServiceEnableDisable(resolve, "disable", []string{"web"}); err != nil {
		t.Fatalf("runServiceEnableDisable() error = %v", err)
	}
}

func TestRunServiceEnablePrintsMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/services":
			writeJSON(w, []client.ServiceStatus{{ID: "svc-1", Name: "web", Status: "stopped", Enabled: false}})
		case "/api/services/enable":
			writeJSON(w, client.ServiceActionResponse{
				Status:  "ok",
				Message: "The server won't start immediately until daemon checks at next time",
				Service: &client.ServiceStatus{ID: "svc-1", Name: "web", Status: "stopped", Enabled: true},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	resolve := func() (*client.Client, error) {
		return client.New(server.URL, ""), nil
	}
	if err := runServiceEnableDisable(resolve, "enable", []string{"web"}); err != nil {
		t.Fatalf("runServiceEnableDisable() error = %v", err)
	}
}

func TestRunServiceUpdatePatchesFieldsWithoutRestart(t *testing.T) {
	var saved client.ServiceDefinition
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/services":
			if r.Method == http.MethodGet {
				writeJSON(w, []client.ServiceStatus{{
					ID:          "svc-1",
					Name:        "web",
					Command:     "old",
					WorkingDir:  "/old",
					ExtraEnv:    map[string]string{"OLD": "1", "KEEP": "yes"},
					PortForward: &client.ServicePortForwardStatus{Port: 8080, Label: "old-label", Provider: "localtunnel"},
				}})
				return
			}
			if r.Method != http.MethodPut || r.URL.Query().Get("restart") != "false" {
				t.Fatalf("save request = %s %s, want PUT /api/services?restart=false", r.Method, r.URL.String())
			}
			if err := json.NewDecoder(r.Body).Decode(&saved); err != nil {
				t.Fatalf("decode save request: %v", err)
			}
			writeJSON(w, client.ServiceStatus{ID: "svc-1", Name: saved.Name, Command: saved.Command})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	resolve := func() (*client.Client, error) {
		return client.New(server.URL, ""), nil
	}
	err := runServiceUpdate(resolve, []string{
		"web",
		"--command", "new",
		"--working-dir", "/new",
		"--env", "A=1",
		"--unset-env", "OLD",
		"--port", "9090",
		"--port-label", "api",
	})
	if err != nil {
		t.Fatalf("runServiceUpdate() error = %v", err)
	}
	if saved.Command != "new" || saved.WorkingDir != "/new" {
		t.Fatalf("saved command/working dir = %q/%q, want new//new", saved.Command, saved.WorkingDir)
	}
	if saved.ExtraEnv["A"] != "1" || saved.ExtraEnv["KEEP"] != "yes" {
		t.Fatalf("saved env = %#v, want A and KEEP", saved.ExtraEnv)
	}
	if _, ok := saved.ExtraEnv["OLD"]; ok {
		t.Fatalf("saved env still has OLD: %#v", saved.ExtraEnv)
	}
	if saved.PortForward == nil || saved.PortForward.Port != 9090 || saved.PortForward.Label != "api" || saved.PortForward.Provider != "localtunnel" {
		t.Fatalf("saved port forward = %#v, want port 9090 label api provider preserved", saved.PortForward)
	}
}
