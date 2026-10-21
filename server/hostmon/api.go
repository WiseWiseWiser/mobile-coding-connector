package hostmon

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// RegisterAPI mounts the host-monitor query and control routes.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/hostmon/status", handleStatus)
	mux.HandleFunc("/api/remote-agent/hostmon/samples", handleSamples)
	mux.HandleFunc("/api/remote-agent/hostmon/start", handleStart)
	mux.HandleFunc("/api/remote-agent/hostmon/stop", handleStop)
	mux.HandleFunc("/api/remote-agent/hostmon/enable", handleEnable)
	mux.HandleFunc("/api/remote-agent/hostmon/disable", handleDisable)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	writeJSON(w, http.StatusOK, Default().Status())
}

func handleSamples(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	n := 20
	if raw := r.URL.Query().Get("lines"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeErr(w, http.StatusBadRequest, "lines must be a positive integer")
			return
		}
		n = parsed
	}
	samples, err := Default().Samples(n)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if samples == nil {
		samples = []Sample{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"samples": samples})
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	handleControl(w, r, func() error { return Default().Start() })
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	handleControl(w, r, func() error { return Default().Stop() })
}

func handleEnable(w http.ResponseWriter, r *http.Request) {
	handleControl(w, r, func() error { return Default().SetAutoStart(true) })
}

func handleDisable(w http.ResponseWriter, r *http.Request) {
	handleControl(w, r, func() error { return Default().SetAutoStart(false) })
}

func handleControl(w http.ResponseWriter, r *http.Request, fn func() error) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	if err := fn(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, Default().Status())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
