package codexweb

import (
	"encoding/json"
	"net/http"
	"time"
)

// StatusResponse represents the server status response
type StatusResponse struct {
	Running   bool      `json:"running"`
	Port      int       `json:"port"`
	Timestamp time.Time `json:"timestamp"`
}

// StartRequest represents the start server request
type StartRequest struct {
	Port int `json:"port,omitempty"`
}

// StartResponse represents the start server response
type StartResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Port    int    `json:"port"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RegisterRoutes registers the codex-web API routes
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/codex-web/status", handleStatus)
	mux.HandleFunc("POST /api/codex-web/start", handleStart)
	mux.HandleFunc("POST /api/codex-web/stop", handleStop)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	manager := GetGlobalManager()

	// Check both if we think it's running and if it responds to HTTP
	running := manager.IsRunning() || manager.CheckServerHTTP()

	response := StatusResponse{
		Running:   running,
		Port:      manager.GetPort(),
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use default port if no request body
		req.Port = 0
	}

	manager := GetGlobalManager()

	// Check if already running
	if manager.IsRunning() || manager.CheckServerHTTP() {
		response := StartResponse{
			Success: true,
			Message: "Codex Web server is already running",
			Port:    manager.GetPort(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Try to start the server
	if err := manager.Start(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := ErrorResponse{
			Error:   "failed_to_start",
			Message: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Wait a moment for server to start
	time.Sleep(2 * time.Second)

	response := StartResponse{
		Success: true,
		Message: "Codex Web server started successfully",
		Port:    manager.GetPort(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	manager := GetGlobalManager()

	if err := manager.Stop(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := ErrorResponse{
			Error:   "failed_to_stop",
			Message: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Codex Web server stopped successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
