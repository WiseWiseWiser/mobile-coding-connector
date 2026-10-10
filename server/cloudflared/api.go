package cloudflared

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/xhd2015/ai-critic/server/cloudflare"
	serverqemu "github.com/xhd2015/ai-critic/server/qemu"
	"github.com/xhd2015/ai-critic/server/streaming/progress"
)

// RegisterAPI mounts remote-agent cloudflared status and use.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/cloudflared/status", handleStatus)
	mux.HandleFunc("/api/remote-agent/cloudflared/status/stream", handleStatusStream)
	mux.HandleFunc("/api/remote-agent/cloudflared/use/stream", handleUseStream)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	snap, err := Observe()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	view := FormatStatus(snap.Files, snap.Runtime)
	writeJSON(w, http.StatusOK, map[string]any{
		"backend":    string(EffectiveBackend(snap.Files, snap.Runtime)),
		"configured": string(ConfiguredBackend(snap.Files)),
		"lines":      view.Lines,
		"warning":    view.Warning,
		"public":     snap.Runtime.PublicHost,
	})
}

func handleStatusStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pw := progress.NewWriter(w)
	if pw == nil {
		writeErr(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	files, public, err := LoadFiles()
	if err != nil {
		_ = pw.EmitError(err.Error())
		return
	}
	RunCloudflaredStatus(r.Context(), files, public, probeGuestCF, probeDials, StatusSink{
		Emit: func(line string) { _ = pw.EmitLog(line, true) },
		Warn: func(message string) { _ = pw.EmitLog("warning: "+message, true) },
	})
	_ = pw.EmitDone(map[string]any{"ok": true})
}

func probeGuestCF(ctx context.Context) GuestCF {
	kv, err := serverqemu.DefaultManager().CFStatusLines(ctx, "", nil)
	if kv["cf_alive"] == "" && err != nil {
		return GuestCF{Err: err}
	}
	return GuestCF{Alive: kv["cf_alive"] == "yes", PID: kv["cf_pid"]}
}

func probeDials(context.Context) Dials {
	counts, err := cloudflare.ProxyDialCounts()
	if err != nil {
		return Dials{Err: err}
	}
	n := 0
	for _, c := range counts {
		n += c
	}
	return Dials{Count: n}
}

func handleUseStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Backend string `json:"backend"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	target, err := ParseBackend(req.Backend)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	pw := progress.NewWriter(w)
	if pw == nil {
		writeErr(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	summary, err := Apply(target, func(msg string) {
		_ = pw.EmitLog(msg, true)
	})
	if err != nil {
		_ = pw.EmitError(err.Error())
		return
	}
	for _, line := range FormatUse(summary) {
		if err := pw.EmitLog(line, true); err != nil {
			return
		}
	}
	_ = pw.EmitDone(map[string]any{
		"backend":   string(summary.Target),
		"published": summary.Published,
		"ping":      summary.Ping,
	})
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
