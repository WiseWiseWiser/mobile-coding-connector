package projectpull

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// RegisterAPI registers pull-local HTTP endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/project/pull-local", handlePullLocal)
	mux.HandleFunc("/api/remote-agent/project/pull-local/truncate", handleTruncate)
	mux.HandleFunc("/api/remote-agent/project/pull-local/inspect", handleInspect)
	mux.HandleFunc("/api/remote-agent/project/pull-local/bundle", handleBundle)
	mux.HandleFunc("/api/remote-agent/project/pull-local/download", handleDownload)
}

func handleInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req InspectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	out, err := InspectRepo(req.Dir)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func handleBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req BundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := validateDir(req.Dir); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	if err := WriteBundle(w, req.Dir); err != nil {
		// Headers may already be sent; best-effort log-style body.
		fmt.Fprintf(w, "error: %v\n", err)
		return
	}
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := validateDir(req.Dir); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/gzip")
	if err := WriteDownloadPackage(w, req.Dir); err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return
	}
}

func handlePullLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req PullLocalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.DryRun {
		plan, err := BuildPlan(req)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(plan)
		return
	}
	if _, err := BuildPlan(req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/gzip")
	if err := WritePackage(w, req); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func handleTruncate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req TruncateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := TruncateWorktree(req.Dir, req.Commit); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}