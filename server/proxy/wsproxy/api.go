package wsproxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/ws-proxy/status", handleStatus)
	mux.HandleFunc("POST /api/ws-proxy/start", handleStart)
	mux.HandleFunc("POST /api/ws-proxy/start/stream", handleStartStream)
	mux.HandleFunc("POST /api/ws-proxy/stop", handleStop)
	mux.HandleFunc("GET /api/ws-proxy/config", handleGetConfig)
	mux.HandleFunc("PUT /api/ws-proxy/config", handlePutConfig)
	mux.HandleFunc("GET /api/ws-proxy/vmess-link", handleVMessLink)
	mux.HandleFunc("GET /api/ws-proxy/doctor", handleDoctor)
}

func handleStartStream(w http.ResponseWriter, r *http.Request) {
	tmp := false
	if r.URL.Query().Get("tmp") == "true" || r.URL.Query().Get("tmp") == "1" {
		tmp = true
	}

	m := GetManager()
	if err := m.StartStream(w, tmp); err != nil {
		writeAPIErr(w, toAPIError(err))
		return
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	m := GetManager()
	if err := m.Recover(); err != nil {
		fmt.Printf("[ws-proxy] status recover: %v\n", err)
	}
	writeJSON(w, m.Status())
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	m := GetManager()

	tmp := false
	if r.URL.Query().Get("tmp") == "true" || r.URL.Query().Get("tmp") == "1" {
		tmp = true
	}

	if r.URL.Query().Get("dry_run") == "true" || r.URL.Query().Get("dry_run") == "1" {
		dr, err := m.DryRun(tmp)
		if err != nil {
			writeAPIErr(w, newError(ErrInternal, err.Error()))
			return
		}
		writeJSON(w, map[string]interface{}{
			"dry_run": dr,
		})
		return
	}

	err := m.Start(tmp)
	if err != nil {
		writeAPIErr(w, toAPIError(err))
		return
	}

	status := m.Status()
	result := map[string]interface{}{
		"running":    status.Running,
		"public_url": status.PublicURL,
		"vmess_link": m.GetVMessLink(),
	}
	writeJSON(w, result)
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	m := GetManager()
	if err := m.Stop(); err != nil {
		writeAPIErr(w, newError(ErrInternal, err.Error()))
		return
	}
	writeJSON(w, map[string]interface{}{
		"running": false,
	})
}

func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := LoadConfig()
	if err != nil {
		writeAPIErr(w, newError(ErrInternal, "failed to load config"))
		return
	}
	if cfg == nil {
		cfg = defaultConfig()
	}
	writeJSON(w, cfg)
}

func handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UpstreamProxy string `json:"upstream_proxy"`
		ListenPort    *int   `json:"listen_port"`
		WSPath        string `json:"ws_path"`
		Subdomain     string `json:"subdomain"`
		AutoStart     *bool  `json:"auto_start"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIErr(w, newError(ErrBadRequest, "invalid request body"))
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		writeAPIErr(w, newError(ErrInternal, "failed to load config"))
		return
	}
	if cfg == nil {
		cfg = defaultConfig()
	}

	if req.UpstreamProxy != "" {
		cfg.UpstreamProxy = req.UpstreamProxy
	}
	if req.ListenPort != nil {
		cfg.ListenPort = *req.ListenPort
	}
	if req.WSPath != "" {
		cfg.WSPath = req.WSPath
	}
	if req.Subdomain != "" {
		cfg.Subdomain = req.Subdomain
	}
	if req.AutoStart != nil {
		cfg.AutoStart = *req.AutoStart
	}

	if err := SaveConfig(cfg); err != nil {
		writeAPIErr(w, newError(ErrInternal, "failed to save config"))
		return
	}

	writeJSON(w, cfg)
}

func handleDoctor(w http.ResponseWriter, r *http.Request) {
	tryURL := r.URL.Query().Get("try_url")
	report := GetManager().Doctor(tryURL)
	writeJSON(w, report)
}

func handleVMessLink(w http.ResponseWriter, r *http.Request) {
	m := GetManager()
	if err := m.Recover(); err != nil {
		fmt.Printf("[ws-proxy] vmess-link recover: %v\n", err)
	}
	if !m.Status().Running {
		writeAPIErr(w, newError(ErrNotRunning, "ws-proxy is not running"))
		return
	}

	link := m.GetVMessLink()
	if link == "" {
		writeAPIErr(w, newError(ErrNotRunning, "ws-proxy is not running or not configured"))
		return
	}

	cfg, err := m.GetVMessConfig()
	if err != nil {
		writeAPIErr(w, newError(ErrNotRunning, err.Error()))
		return
	}

	writeJSON(w, map[string]interface{}{
		"vmess_link": link,
		"host":       cfg.Host,
		"port":       cfg.Port,
		"uuid":       cfg.UUID,
		"alter_id":   cfg.AlterID,
		"network":    cfg.Network,
		"type":       cfg.Type,
		"path":       cfg.Path,
		"tls":        cfg.TLS,
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeAPIErr(w http.ResponseWriter, apiErr *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(apiErr.Code))
	json.NewEncoder(w).Encode(map[string]interface{}{"error": apiErr})
}

func httpStatusFor(code ErrorCode) int {
	switch code {
	case ErrBadRequest:
		return http.StatusBadRequest
	case ErrNotRunning:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func toAPIError(err error) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}
	msg := err.Error()

	if strings.Contains(msg, "upstream_proxy is not configured") ||
		strings.Contains(msg, "upstream_proxy is not set") {
		return newError(ErrNotConfigured, "upstream_proxy is not configured")
	}
	if strings.Contains(msg, "already running") {
		return newError(ErrAlreadyRunning, "ws-proxy is already running")
	}
	if strings.Contains(msg, "no domain configured") ||
		strings.Contains(msg, "base_domain") {
		return newError(ErrNoDomain, "no domain configured")
	}
	if strings.Contains(msg, "already in use") {
		return newError(ErrPortInUse, msg)
	}
	if strings.Contains(msg, "failed to start xray") ||
		strings.Contains(msg, "failed to become healthy") {
		return newError(ErrStartupFailed, msg)
	}
	if strings.Contains(msg, "failed to start cloudflared") ||
		strings.Contains(msg, "cloudflared error") ||
		strings.Contains(msg, "timeout waiting") {
		return newError(ErrTunnelFailed, msg)
	}
	return newError(ErrInternal, msg)
}
