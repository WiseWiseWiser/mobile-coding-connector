package machineanalyse

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// RegisterAPI registers machine analyse-files HTTP endpoints that use the
// process HOME environment variable as the scan root (production path).
func RegisterAPI(mux *http.ServeMux) {
	RegisterAPIForHome(mux, "")
}

// RegisterAPIForHome registers the same endpoints as RegisterAPI, but scopes
// analyse-files scanning to the given home directory. When home is empty,
// handlers fall back to os.Getenv("HOME") like RegisterAPI.
//
// Intended for in-process tests that want an explicit home without mutating
// process environment.
func RegisterAPIForHome(mux *http.ServeMux, home string) {
	homeFn := func() string {
		if strings.TrimSpace(home) != "" {
			return home
		}
		return os.Getenv("HOME")
	}
	mux.HandleFunc("/api/remote-agent/machine/analyse-files/stream", func(w http.ResponseWriter, r *http.Request) {
		handleAnalyseFilesStream(w, r, homeFn())
	})
}

func handleAnalyseFilesStream(w http.ResponseWriter, r *http.Request, home string) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Accept empty or {} body for forward compatibility.
	if r.Body != nil && r.ContentLength != 0 {
		var discard map[string]any
		_ = json.NewDecoder(r.Body).Decode(&discard)
	}
	if err := AnalyseFilesStream(w, home); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
