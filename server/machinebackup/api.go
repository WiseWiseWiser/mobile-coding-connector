package machinebackup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// RegisterAPI registers machine backup/restore HTTP endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/machine/backup", handleBackup)
	mux.HandleFunc("/api/remote-agent/machine/backup/stream", handleBackupStream)
	mux.HandleFunc("/api/remote-agent/machine/backup/archive", handleBackupArchiveDownload)
	mux.HandleFunc("/api/remote-agent/machine/restore", handleRestore)
	mux.HandleFunc("/api/remote-agent/machine/restore/stream", handleRestoreStream)
	mux.HandleFunc("/api/remote-agent/machine/config", handleBuiltinConfig)
	mux.HandleFunc("/api/remote-agent/machine/backup-config", handleBackupConfig)
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

func handleBackupConfig(w http.ResponseWriter, r *http.Request) {
	home := os.Getenv("HOME")
	switch r.Method {
	case http.MethodGet:
		exclude, include := parsePathRulesQuery(r)
		largeDirThreshold := strings.TrimSpace(r.URL.Query().Get("large_dir_threshold"))
		cfg, err := EffectiveExclusionConfigWithOverrides(home, exclude, include, largeDirThreshold)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "invalid large_dir_threshold") {
				status = http.StatusBadRequest
			}
			writeJSONError(w, status, err.Error())
			return
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(append(data, '\n'))
	case http.MethodPut:
		var req BackupConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
		if req.Exclude == nil {
			req.Exclude = []string{}
		}
		threshold := strings.TrimSpace(req.LargeDirThreshold)
		if len(req.Exclude) == 0 && threshold == "" {
			writeJSONError(w, http.StatusBadRequest, "backup config requires exclude paths and/or large_dir_threshold")
			return
		}
		entries := ExcludePathsFromStrings(req.Exclude)
		if err := SaveUserBackupConfig(home, entries, threshold); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		cfg, err := EffectiveExclusionConfigForHome(home)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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
	gitOpts := GitScanOptions{
		SkipGitDirsScan:     req.SkipGitDirsScan,
		GitDirsScanMaxDepth: req.GitDirsScanMaxDepth,
	}
	if req.DryRun {
		plan, err := BuildPlan(home, req.Exclude, req.Include, gitOpts)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(plan)
		return
	}

	w.Header().Set("Content-Type", "application/x-xz")
	if err := WriteArchive(w, home, req.Exclude, req.Include, gitOpts); err != nil {
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
	home := os.Getenv("HOME")
	thresholdBytes, err := ResolveLargeDirThresholdBytes(home, req.LargeDirThresholdBytes)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	gitOpts := GitScanOptions{
		SkipGitDirsScan:     req.SkipGitDirsScan,
		GitDirsScanMaxDepth: req.GitDirsScanMaxDepth,
	}
	if err := BackupStream(w, home, req.Exclude, req.Include, BackupStreamOptions{
		LargeDirThresholdBytes: thresholdBytes,
		GitOpts:                gitOpts,
		WriteArchive:           req.Archive,
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func handleBackupArchiveDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	body, err := openArchiveSession(token)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "application/x-xz")
	if _, err := io.Copy(w, body); err != nil {
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