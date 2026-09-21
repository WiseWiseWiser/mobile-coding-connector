package gomod

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ActionRequest is the JSON body for gomod lifecycle actions.
type ActionRequest struct {
	Now   bool   `json:"now,omitempty"`
	Port  int    `json:"port,omitempty"`
	Root  string `json:"root,omitempty"`
	Lines int    `json:"lines,omitempty"`
}

// ActionResponse is a simple JSON envelope for CLI friendliness.
type ActionResponse struct {
	OK     bool              `json:"ok"`
	KV     map[string]string `json:"kv,omitempty"`
	Output string            `json:"output,omitempty"`
	Error  string            `json:"error,omitempty"`
}

// RegisterAPI mounts gomod remote-agent endpoints under /api/remote-agent/gomod/.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/gomod/", handleGomodPath)
}

func handleGomodPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeActionErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	action := strings.TrimPrefix(r.URL.Path, "/api/remote-agent/gomod/")
	action = strings.Trim(action, "/")
	if action == "" {
		writeActionErr(w, http.StatusNotFound, "not found")
		return
	}
	var req ActionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	applyQueryFlags(r, &req)

	m, err := DefaultManager()
	if err != nil {
		writeActionErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	switch action {
	case "status":
		writeAction(w, ActionResponse{OK: true, KV: m.Status()})
	case "start":
		if req.Root != "" {
			if err := m.SetRoot(req.Root); err != nil {
				writeActionErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if req.Port > 0 {
			if err := m.SetPort(req.Port); err != nil {
				writeActionErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		kv, err := m.Start()
		writeKvOrErr(w, kv, err, "")
	case "stop":
		kv, err := m.Stop()
		if err == nil && kv == nil {
			writeAction(w, ActionResponse{OK: true, Output: "warning: mod-proxy is not running"})
			return
		}
		writeKvOrErr(w, kv, err, "")
	case "enable":
		if req.Root != "" {
			if err := m.SetRoot(req.Root); err != nil {
				writeActionErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if req.Port > 0 {
			if err := m.SetPort(req.Port); err != nil {
				writeActionErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		kv, err := m.Enable(req.Now)
		writeKvOrErr(w, kv, err, "")
	case "disable":
		kv, err := m.Disable()
		writeKvOrErr(w, kv, err, "")
	case "logs":
		out, err := m.TailLog(req.Lines)
		if err != nil {
			writeActionErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeAction(w, ActionResponse{OK: true, Output: out})
	default:
		writeActionErr(w, http.StatusNotFound, fmt.Sprintf("unknown gomod action %q", action))
	}
}

func applyQueryFlags(r *http.Request, req *ActionRequest) {
	q := r.URL.Query()
	if v := q.Get("now"); v == "1" || v == "true" {
		req.Now = true
	}
	if v := q.Get("port"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.Port = n
		}
	}
	if v := q.Get("root"); v != "" {
		req.Root = v
	}
	if v := q.Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.Lines = n
		}
	}
}

func writeKvOrErr(w http.ResponseWriter, kv map[string]string, err error, _ string) {
	if err != nil {
		writeActionErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeAction(w, ActionResponse{OK: true, KV: kv})
}

func writeAction(w http.ResponseWriter, resp ActionResponse) {
	writeJSON(w, http.StatusOK, resp)
}

func writeActionErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, ActionResponse{OK: false, Error: msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
