package procmon

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// RegisterAPI mounts GET /api/server/processes and POST /api/server/processes/kill.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/server/processes", handleProcesses)
	mux.HandleFunc("/api/server/processes/kill", handleKill)
}

func handleProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap, err := Get()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.URL.Query().Get("summary") == "1" {
		copySnap := *snap
		copySnap.Processes = []Process{}
		snap = &copySnap
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func handleKill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
	if err != nil || pid < 0 {
		http.Error(w, "invalid pid", http.StatusBadRequest)
		return
	}
	result, err := KillProcess(pid, r.URL.Query().Get("signal"))
	if err != nil {
		http.Error(w, err.Error(), killStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
