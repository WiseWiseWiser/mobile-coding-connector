package qemu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	shared "github.com/xhd2015/dot-pkgs/go-pkgs/qemu"
)

// ActionRequest is the JSON body for qemu lifecycle / cloudflared actions.
type ActionRequest struct {
	DryRun  bool   `json:"dry_run,omitempty"`
	Yes     bool   `json:"yes,omitempty"`
	Deep    bool   `json:"deep,omitempty"`
	Lines   int    `json:"lines,omitempty"`
	URL     string `json:"url,omitempty"`
	Origin  string `json:"origin,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
	Argv    []string `json:"argv,omitempty"`
}

// ActionResponse is a simple JSON envelope for CLI friendliness.
type ActionResponse struct {
	OK     bool              `json:"ok"`
	KV     map[string]string `json:"kv,omitempty"`
	Output string            `json:"output,omitempty"`
	Error  string            `json:"error,omitempty"`
}

// ConfigResponse is GET/PUT /api/remote-agent/qemu/config.
type ConfigResponse struct {
	Enabled bool `json:"enabled"`
}

// RegisterAPI mounts qemu remote-agent endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/qemu/config", handleConfig)
	mux.HandleFunc("/api/remote-agent/qemu/", handleQemuPath)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := LoadConfig()
		if err != nil {
			writeActionErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, ConfigResponse{Enabled: cfg.Enabled})
	case http.MethodPut:
		var req ConfigResponse
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeActionErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := SaveConfig(shared.FileConfig{Enabled: req.Enabled}); err != nil {
			writeActionErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, ConfigResponse{Enabled: req.Enabled})
	default:
		writeActionErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleQemuPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeActionErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/remote-agent/qemu/")
	path = strings.Trim(path, "/")
	if path == "" || path == "config" {
		writeActionErr(w, http.StatusNotFound, "not found")
		return
	}

	var req ActionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	applyQueryFlags(r, &req)

	if strings.HasPrefix(path, "cloudflared/") {
		action := strings.TrimPrefix(path, "cloudflared/")
		resp := runCFAction(action, req)
		writeAction(w, resp)
		return
	}

	switch path {
	case "exec":
		resp := runExecAction(req)
		writeAction(w, resp)
	default:
		resp := runGuestAction(path, req)
		writeAction(w, resp)
	}
}

func applyQueryFlags(r *http.Request, req *ActionRequest) {
	q := r.URL.Query()
	if v := q.Get("dry_run"); v == "1" || v == "true" {
		req.DryRun = true
	}
	if v := q.Get("yes"); v == "1" || v == "true" {
		req.Yes = true
	}
	if v := q.Get("deep"); v == "1" || v == "true" {
		req.Deep = true
	}
	if v := q.Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.Lines = n
		}
	}
	if v := q.Get("url"); v != "" {
		req.URL = v
	}
	if v := q.Get("origin"); v != "" {
		req.Origin = v
	}
	if v := q.Get("timeout"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.Timeout = n
		}
	}
}

func originOf(req ActionRequest) string {
	if req.Origin != "" {
		return req.Origin
	}
	return req.URL
}

func newManager(req ActionRequest, stdout, stderr io.Writer) *shared.Manager {
	m := DefaultManager()
	m.DryRun = req.DryRun
	m.Stdout = stdout
	m.Stderr = stderr
	return m
}

func runGuestAction(action string, req ActionRequest) ActionResponse {
	var outBuf, errBuf bytes.Buffer
	m := newManager(req, &outBuf, &errBuf)

	switch action {
	case "status":
		kv, err := m.Status()
		return actionFromKV(kv, outBuf.String(), err)
	case "start":
		kv, err := m.Start()
		return actionFromKV(kv, outBuf.String(), err)
	case "stop":
		err := m.Stop()
		return actionFromErr(outBuf.String(), err)
	case "restart":
		kv, err := m.Restart()
		return actionFromKV(kv, outBuf.String(), err)
	case "purge":
		if !req.Yes && !req.DryRun {
			return ActionResponse{OK: false, Error: "purge requires yes=true"}
		}
		err := m.Purge(req.Deep)
		return actionFromErr(outBuf.String(), err)
	case "logs":
		lines := req.Lines
		if lines <= 0 {
			lines = 80
		}
		out, err := m.Logs(lines)
		if err != nil {
			return ActionResponse{OK: false, Output: mergeOut(outBuf.String(), out), Error: err.Error()}
		}
		return ActionResponse{OK: true, Output: mergeOut(outBuf.String(), out)}
	case "show":
		return ActionResponse{OK: true, Output: showText(m.Cfg)}
	case "doctor":
		kv, err := m.Status()
		var b strings.Builder
		b.WriteString(formatStatus(kv))
		lines := req.Lines
		if lines <= 0 {
			lines = 40
		}
		if !req.DryRun {
			if logs, lerr := m.Logs(lines); lerr == nil {
				b.WriteString("serial:\n")
				b.WriteString(logs)
				if !strings.HasSuffix(logs, "\n") {
					b.WriteByte('\n')
				}
			} else {
				fmt.Fprintf(&b, "warning: serial: %v\n", lerr)
			}
		}
		b.WriteString(outBuf.String())
		b.WriteString("hint: enable guest CF tunnels with {\"enabled\":true} in qemu.json\n")
		b.WriteString("hint: guest state under ~/.ai-critic/qemu\n")
		if err != nil {
			return ActionResponse{OK: false, KV: kv, Output: b.String(), Error: err.Error()}
		}
		ok := kv["qemu_alive"] == "yes"
		resp := ActionResponse{OK: ok, KV: kv, Output: b.String()}
		if !ok {
			resp.Error = "qemu: down"
		}
		return resp
	default:
		return ActionResponse{OK: false, Error: fmt.Sprintf("unknown action %q", action)}
	}
}

func runCFAction(action string, req ActionRequest) ActionResponse {
	var outBuf, errBuf bytes.Buffer
	m := newManager(req, &outBuf, &errBuf)
	origin := originOf(req)

	switch action {
	case "status":
		kv, err := m.CFStatus(origin)
		return actionFromKV(kv, outBuf.String(), err)
	case "start":
		kv, err := m.CFStart(origin)
		return actionFromKV(kv, outBuf.String(), err)
	case "stop":
		err := m.CFStop()
		return actionFromErr(outBuf.String(), err)
	case "restart":
		kv, err := m.CFRestart(origin)
		return actionFromKV(kv, outBuf.String(), err)
	case "purge":
		if !req.Yes && !req.DryRun {
			return ActionResponse{OK: false, Error: "purge requires yes=true"}
		}
		err := m.CFPurge(req.Deep)
		return actionFromErr(outBuf.String(), err)
	case "logs":
		lines := req.Lines
		if lines <= 0 {
			lines = 80
		}
		out, err := m.CFLogs(lines)
		if err != nil {
			return ActionResponse{OK: false, Output: mergeOut(outBuf.String(), out), Error: err.Error()}
		}
		return ActionResponse{OK: true, Output: mergeOut(outBuf.String(), out)}
	case "login":
		kv, err := m.CFLogin()
		return actionFromKV(kv, outBuf.String(), err)
	case "list":
		out, err := m.CFList()
		if err != nil {
			return ActionResponse{OK: false, Output: mergeOut(outBuf.String(), out), Error: err.Error()}
		}
		return ActionResponse{OK: true, Output: mergeOut(outBuf.String(), out)}
	case "show":
		return ActionResponse{OK: true, Output: showCFText(m.Cfg, origin)}
	case "doctor":
		kv, err := m.CFStatus(origin)
		var b strings.Builder
		b.WriteString(formatCFStatus(kv))
		lines := req.Lines
		if lines <= 0 {
			lines = 40
		}
		if !req.DryRun {
			if logs, lerr := m.CFLogs(lines); lerr == nil {
				b.WriteString("log:\n")
				b.WriteString(logs)
				if !strings.HasSuffix(logs, "\n") {
					b.WriteByte('\n')
				}
			}
		}
		b.WriteString(outBuf.String())
		if err != nil {
			return ActionResponse{OK: false, KV: kv, Output: b.String(), Error: err.Error()}
		}
		ok := kv["cf_alive"] == "yes"
		resp := ActionResponse{OK: ok, KV: kv, Output: b.String()}
		if !ok {
			resp.Error = "cloudflared: down"
		}
		return resp
	case "named-start":
		// body: argv = [tunnelName, hostname...]
		if len(req.Argv) < 2 {
			return ActionResponse{OK: false, Error: "named-start requires argv: [tunnelName, hostname...]"}
		}
		name := req.Argv[0]
		hosts := req.Argv[1:]
		kv, err := m.CFNamedStart(name, origin, hosts)
		return actionFromKV(kv, outBuf.String(), err)
	default:
		return ActionResponse{OK: false, Error: fmt.Sprintf("unknown cloudflared action %q", action)}
	}
}

func runExecAction(req ActionRequest) ActionResponse {
	if len(req.Argv) == 0 {
		return ActionResponse{OK: false, Error: "exec requires argv"}
	}
	cfg := shared.AiCriticConfig()
	sshArgs := shared.GuestSSHArgs(cfg, false)
	// GuestSSHArgs returns ["ssh", ...flags..., "user@host"]; append remote command.
	full := append(sshArgs, req.Argv...)
	if req.DryRun {
		return ActionResponse{OK: true, Output: "[dry-run] would " + strings.Join(full, " ") + "\n"}
	}
	cmd := exec.Command(full[0], full[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ActionResponse{OK: false, Output: string(out), Error: err.Error()}
	}
	return ActionResponse{OK: true, Output: string(out)}
}

func actionFromKV(kv map[string]string, dryOut string, err error) ActionResponse {
	out := dryOut
	if out == "" && len(kv) > 0 {
		out = formatKVLines(kv)
	}
	if err != nil {
		return ActionResponse{OK: false, KV: kv, Output: out, Error: err.Error()}
	}
	return ActionResponse{OK: true, KV: kv, Output: out}
}

func actionFromErr(dryOut string, err error) ActionResponse {
	if err != nil {
		return ActionResponse{OK: false, Output: dryOut, Error: err.Error()}
	}
	return ActionResponse{OK: true, Output: dryOut}
}

func mergeOut(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + b
	}
}

func formatKVLines(kv map[string]string) string {
	var b strings.Builder
	for k, v := range kv {
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	return b.String()
}

func formatStatus(kv map[string]string) string {
	var b strings.Builder
	alive := kv["qemu_alive"] == "yes" || kv["skip"] == "running"
	if alive {
		if pid := kv["qemu_pid"]; pid != "" {
			fmt.Fprintf(&b, "qemu:      running  pid=%s\n", pid)
		} else {
			b.WriteString("qemu:      running\n")
		}
	} else {
		b.WriteString("qemu:      down\n")
	}
	cfg := shared.AiCriticConfig()
	fmt.Fprintf(&b, "ssh:       127.0.0.1:%d\n", cfg.SSHPort)
	switch kv["guest_ssh"] {
	case "ok":
		fmt.Fprintf(&b, "guest:     ssh ok  %s@qemu-guest\n", cfg.User)
	case "wait":
		b.WriteString("guest:     ssh wait (booting)\n")
	default:
		b.WriteString("guest:     ssh fail\n")
	}
	fmt.Fprintf(&b, "disk:      overlay %s  backing %s  %s\n",
		shared.HumanBytes(kv["overlay_bytes"]), shared.HumanBytes(kv["backing_bytes"]), cfg.Dir)
	return b.String()
}

func formatCFStatus(kv map[string]string) string {
	var b strings.Builder
	if kv["cf_alive"] == "yes" {
		fmt.Fprintf(&b, "cloudflared: running  pid=%s\n", kv["cf_pid"])
	} else {
		b.WriteString("cloudflared: down\n")
	}
	if u := kv["url"]; u != "" {
		fmt.Fprintf(&b, "url:         %s\n", u)
	}
	if c := kv["cert"]; c != "" {
		fmt.Fprintf(&b, "cert:        %s\n", c)
	}
	return b.String()
}

func showText(_ shared.Config) string {
	c := shared.AiCriticConfig()
	var b strings.Builder
	b.WriteString("kind:       ai-critic qemu guest\n")
	fmt.Fprintf(&b, "dir:        %s\n", c.Dir)
	fmt.Fprintf(&b, "user:       %s\n", c.User)
	fmt.Fprintf(&b, "ssh:        127.0.0.1:%d\n", c.SSHPort)
	fmt.Fprintf(&b, "accel:      %s\n", c.Accel)
	fmt.Fprintf(&b, "mem/cpus:   %dMB / %d\n", c.MemMB, c.CPUs)
	fmt.Fprintf(&b, "backing:    %s\n", c.BackingName)
	fmt.Fprintf(&b, "enabled:    %v  (%s)\n", Enabled(), getConfigFile())
	return b.String()
}

func showCFText(_ shared.Config, origin string) string {
	c := shared.AiCriticConfig()
	if origin == "" {
		origin = c.CFOriginURL
	}
	var b strings.Builder
	b.WriteString("kind:       guest cloudflared\n")
	fmt.Fprintf(&b, "origin:     %s\n", origin)
	fmt.Fprintf(&b, "bin:        %s\n", c.CFBin)
	fmt.Fprintf(&b, "cert:       %s\n", c.CFCert)
	fmt.Fprintf(&b, "log:        %s\n", c.CFLog)
	fmt.Fprintf(&b, "pid:        %s\n", c.CFPid)
	fmt.Fprintf(&b, "guest dir:  %s\n", c.Dir)
	return b.String()
}

func writeAction(w http.ResponseWriter, resp ActionResponse) {
	code := http.StatusOK
	if !resp.OK && resp.Error != "" {
		code = http.StatusBadRequest
	}
	writeJSON(w, code, resp)
}

func writeActionErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, ActionResponse{OK: false, Error: msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
