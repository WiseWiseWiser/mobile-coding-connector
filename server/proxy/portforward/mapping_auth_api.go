package portforward

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/xhd2015/ai-critic/server/config"
)

func handlePortMappingAuth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		portStr := r.URL.Query().Get("port")
		if portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				http.Error(w, "invalid port number", http.StatusBadRequest)
				return
			}
			entry, err := config.GetPortMappingAuth(port)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if entry == nil {
				json.NewEncoder(w).Encode(map[string]any{
					"port": portStr,
					"auth": config.PortAuth{Required: false},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"port": portStr,
				"auth": entry,
			})
			return
		}
		mappings, err := config.LoadPortMappingAuth()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mappings)
	case http.MethodDelete:
		portStr := r.URL.Query().Get("port")
		if portStr == "" {
			http.Error(w, "port parameter is required", http.StatusBadRequest)
			return
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			http.Error(w, "invalid port number", http.StatusBadRequest)
			return
		}
		if err := config.DeletePortMappingAuth(port); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
