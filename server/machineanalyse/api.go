package machineanalyse

import (
	"encoding/json"
	"net/http"
	"os"
)

// RegisterAPI registers machine analyse-files HTTP endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/machine/analyse-files/stream", handleAnalyseFilesStream)
}

func handleAnalyseFilesStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Accept empty or {} body for forward compatibility.
	if r.Body != nil && r.ContentLength != 0 {
		var discard map[string]any
		_ = json.NewDecoder(r.Body).Decode(&discard)
	}
	if err := AnalyseFilesStream(w, os.Getenv("HOME")); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}