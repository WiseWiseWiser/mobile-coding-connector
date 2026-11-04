package httpproxy

import (
	"encoding/json"
	"net/http"
)

// RegisterAPI mounts the JSON endpoints the Services detail page consumes.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/http-proxy/status", handleStatus)
	mux.HandleFunc("POST /api/http-proxy/start", handleStart)
	mux.HandleFunc("POST /api/http-proxy/stop", handleStop)
	mux.HandleFunc("GET /api/http-proxy/config", handleGetConfig)
	mux.HandleFunc("PUT /api/http-proxy/config", handlePutConfig)
	mux.HandleFunc("POST /api/http-proxy/rotate-password", handleRotatePassword)
	mux.HandleFunc("GET /api/http-proxy/connections", handleConnections)
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, GetManager().Status())
}

func handleStart(w http.ResponseWriter, _ *http.Request) {
	m := GetManager()
	if err := m.Start(); err != nil {
		writeAPIErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, m.Status())
}

func handleStop(w http.ResponseWriter, _ *http.Request) {
	if err := GetManager().Stop(); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, GetManager().Status())
}

func handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	cfg, err := LoadConfig()
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	normalized := *cfg
	normalized.normalized()
	writeJSON(w, &normalized)
}

// handlePutConfig updates persisted settings. Changing port or domain while
// running requires a stop/start cycle, which the caller does explicitly.
func handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ListenPort *int    `json:"listen_port"`
		Username   *string `json:"username"`
		Password   *string `json:"password"`
		Domain     *string `json:"domain"`
		AutoStart  *bool   `json:"auto_start"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cfg, err := LoadConfig()
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.ListenPort != nil && *req.ListenPort >= 0 && *req.ListenPort <= 65535 {
		cfg.ListenPort = *req.ListenPort
	}
	if req.Username != nil && *req.Username != "" {
		cfg.Username = *req.Username
	}
	if req.Password != nil && *req.Password != "" {
		cfg.Password = *req.Password
	}
	if req.Domain != nil {
		cfg.Domain = *req.Domain
	}
	if req.AutoStart != nil {
		cfg.AutoStart = *req.AutoStart
	}
	if err := SaveConfig(cfg); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.normalized()
	writeJSON(w, cfg)
}

func handleRotatePassword(w http.ResponseWriter, _ *http.Request) {
	password, err := GetManager().RotatePassword()
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"password": password})
}

func handleConnections(w http.ResponseWriter, _ *http.Request) {
	items, active, connectActive, total, failures := GetManager().Connections()
	writeJSON(w, map[string]interface{}{
		"items":          items,
		"active":         active,
		"connect_active": connectActive,
		"total":          total,
		"failures":       failures,
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": msg})
}
