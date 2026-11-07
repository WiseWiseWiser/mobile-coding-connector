package diskmon

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// defaultAnalysis is the process-wide manager. It is separate from the 5s disk
// snapshot endpoint: scans are manual, slow, and persisted.
var defaultAnalysis = newAnalysisManager(newAnalysisStore(defaultAnalysisDir()))

func registerAnalysisAPI(mux *http.ServeMux) {
	(&analysisAPI{manager: defaultAnalysis}).register(mux)
}

func (a *analysisAPI) register(mux *http.ServeMux) {
	mux.HandleFunc("/api/server/disk/analysis/status", a.handleStatus)
	mux.HandleFunc("/api/server/disk/analysis/start", a.handleStart)
	mux.HandleFunc("/api/server/disk/analysis/cancel", a.handleCancel)
	mux.HandleFunc("/api/server/disk/analysis/reports", a.handleReports)
	mux.HandleFunc("/api/server/disk/analysis/reports/", a.handleReport)
}

type analysisAPI struct {
	manager *analysisManager
}

func (a *analysisAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, a.manager.Status())
}

func (a *analysisAPI) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	job, err := a.manager.Start()
	if errors.Is(err, errAnalysisRunning) {
		// 409 tells the page to attach to the running scan instead of retrying.
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "an analysis is already running",
			"job":   job,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (a *analysisAPI) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cancelled": a.manager.Cancel()})
}

func (a *analysisAPI) handleReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	limit := analysisMaxReports
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	meta, reports, count := a.manager.Reports(limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"reports":           reports,
		"count":             count,
		"last_completed_at": meta.LastCompletedAt,
		"last_attempt":      meta.LastAttempt,
	})
}

func (a *analysisAPI) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	raw := strings.TrimPrefix(r.URL.Path, "/api/server/disk/analysis/reports/")
	id, err := url.PathUnescape(strings.TrimSuffix(raw, "/"))
	if err != nil || id == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "report not found"})
		return
	}
	report, ok := a.manager.Report(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "report not found"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
