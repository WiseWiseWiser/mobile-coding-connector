package exposedurls

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RegisterAPI registers the exposed URLs API endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/exposed-urls", handleList)
	mux.HandleFunc("/api/exposed-urls/add", handleAdd)
	mux.HandleFunc("/api/exposed-urls/update", handleUpdate)
	mux.HandleFunc("/api/exposed-urls/delete", handleDelete)
	mux.HandleFunc("/api/exposed-urls/status", handleStatus)
	mux.HandleFunc("/api/exposed-urls/tunnel/start", handleTunnelStart)
	mux.HandleFunc("/api/exposed-urls/tunnel/stop", handleTunnelStop)
}

// Request/Response types
type addRequest struct {
	ExternalDomain string `json:"external_domain"`
	InternalURL    string `json:"internal_url"`
}

type updateRequest struct {
	ID             string `json:"id"`
	ExternalDomain string `json:"external_domain"`
	InternalURL    string `json:"internal_url"`
}

type deleteRequest struct {
	ID string `json:"id"`
}

type tunnelRequest struct {
	ID string `json:"id"`
}

type cloudflareStatusResponse struct {
	Installed     bool   `json:"installed"`
	Authenticated bool   `json:"authenticated"`
	Error         string `json:"error,omitempty"`
}

func handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	manager := GetManager()
	urls := manager.List()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(urls)
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req addRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ExternalDomain == "" || req.InternalURL == "" {
		http.Error(w, "external_domain and internal_url are required", http.StatusBadRequest)
		return
	}

	manager := GetManager()
	url, err := manager.Add(req.ExternalDomain, req.InternalURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(url)
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if req.ExternalDomain == "" || req.InternalURL == "" {
		http.Error(w, "external_domain and internal_url are required", http.StatusBadRequest)
		return
	}

	manager := GetManager()
	url, err := manager.Update(req.ID, req.ExternalDomain, req.InternalURL)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(url)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req deleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	manager := GetManager()
	if err := manager.Remove(req.ID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	manager := GetManager()
	installed, authenticated, err := manager.CheckCloudflareStatus()

	resp := cloudflareStatusResponse{
		Installed:     installed,
		Authenticated: authenticated,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	manager := GetManager()
	if err := manager.StartTunnel(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func handleTunnelStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	manager := GetManager()
	if err := manager.StopTunnel(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}
