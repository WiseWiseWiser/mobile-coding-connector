package machinebackup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

// RegisterAPI registers machine backup/restore HTTP endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/machine/backup", handleBackup)
	mux.HandleFunc("/api/remote-agent/machine/backup/stream", handleBackupStream)
	mux.HandleFunc("/api/remote-agent/machine/restore", handleRestore)
	mux.HandleFunc("/api/remote-agent/machine/restore/stream", handleRestoreStream)
	mux.HandleFunc("/api/remote-agent/machine/config", handleBuiltinConfig)
}

func handleBuiltinConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	data, err := BuiltinExclusionConfigJSON()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.Exclude == nil {
		req.Exclude = []string{}
	}
	if req.Include == nil {
		req.Include = []string{}
	}

	home := os.Getenv("HOME")
	if req.DryRun {
		plan, err := BuildPlan(home, req.Exclude, req.Include)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(plan)
		return
	}

	w.Header().Set("Content-Type", "application/x-xz")
	if err := WriteArchive(w, home, req.Exclude, req.Include); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func handleBackupStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req BackupStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.Exclude == nil {
		req.Exclude = []string{}
	}
	if req.Include == nil {
		req.Include = []string{}
	}
	if err := BackupPlanStream(w, os.Getenv("HOME"), req.Exclude, req.Include); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	dryRun, err := parseDryRunQuery(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	exclude, include := parsePathRulesQuery(r)

	home := os.Getenv("HOME")
	var plan *MachineRestorePlan
	if dryRun {
		plan, err = BuildRestorePlan(home, r.Body, exclude, include)
	} else {
		plan, err = ApplyRestore(home, r.Body, exclude, include)
	}
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

func handleRestoreStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	dryRun, err := parseDryRunQuery(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	exclude, include := parsePathRulesQuery(r)
	if err := RestorePlanStream(w, os.Getenv("HOME"), r.Body, exclude, include, dryRun); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func parseDryRunQuery(r *http.Request) (bool, error) {
	raw := r.URL.Query().Get("dry_run")
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func parsePathRulesQuery(r *http.Request) (exclude, include []string) {
	q := r.URL.Query()
	exclude = append([]string(nil), q["exclude"]...)
	include = append([]string(nil), q["include"]...)
	return exclude, include
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}