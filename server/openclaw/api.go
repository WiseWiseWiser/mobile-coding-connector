package openclaw

import (
	"encoding/json"
	"net/http"
	"strings"
)

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/openclaw/status", handleStatus)
	mux.HandleFunc("POST /api/openclaw/start", handleStart)
	mux.HandleFunc("POST /api/openclaw/stop", handleStop)
	mux.HandleFunc("GET /api/openclaw/config", handleGetConfig)
	mux.HandleFunc("PUT /api/openclaw/config", handlePutConfig)
	mux.HandleFunc("GET /api/openclaw/doctor", handleDoctor)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, GetManager().Status())
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	m := GetManager()

	if r.URL.Query().Get("dry_run") == "true" || r.URL.Query().Get("dry_run") == "1" {
		dr, err := m.DryRun()
		if err != nil {
			writeAPIErr(w, toAPIError(err))
			return
		}
		writeJSON(w, map[string]any{"dry_run": dr})
		return
	}

	if err := m.Start(); err != nil {
		writeAPIErr(w, toAPIError(err))
		return
	}

	status := m.Status()
	writeJSON(w, map[string]any{
		"running":          status.Running,
		"mocked":           status.Mocked,
		"mock_pid":         status.MockPID,
		"gateway_port":     status.GatewayPort,
		"generated_config": status.GeneratedConfig,
	})
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if err := GetManager().Stop(); err != nil {
		writeAPIErr(w, newError(ErrInternal, err.Error()))
		return
	}
	writeJSON(w, map[string]any{"running": false, "mocked": true})
}

func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := LoadConfig()
	if err != nil {
		writeAPIErr(w, newError(ErrInternal, "failed to load config"))
		return
	}
	writeJSON(w, MaskConfig(cfg))
}

func handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIErr(w, newError(ErrBadRequest, "invalid request body"))
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		writeAPIErr(w, newError(ErrInternal, "failed to load config"))
		return
	}

	merged := MergeConfig(cfg, req)
	if err := SaveConfig(merged); err != nil {
		writeAPIErr(w, newError(ErrInternal, "failed to save config"))
		return
	}
	writeJSON(w, MaskConfig(merged))
}

func handleDoctor(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, GetManager().Doctor())
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIErr(w http.ResponseWriter, apiErr *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(apiErr.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": apiErr})
}

func httpStatusFor(code ErrorCode) int {
	switch code {
	case ErrBadRequest:
		return http.StatusBadRequest
	case ErrNotRunning:
		return http.StatusServiceUnavailable
	case ErrAlreadyRunning:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func toAPIError(err error) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}
	msg := err.Error()
	if strings.Contains(msg, "already running") {
		return newError(ErrAlreadyRunning, "openclaw gateway is already running")
	}
	if strings.Contains(msg, "slack bot token") ||
		strings.Contains(msg, "slack app token") ||
		strings.Contains(msg, "only socket mode") {
		return newError(ErrBadRequest, msg)
	}
	return newError(ErrInternal, msg)
}