package cursorweb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

// StatusResponse represents server status.
type StatusResponse struct {
	Running   bool      `json:"running"`
	Port      int       `json:"port"`
	Timestamp time.Time `json:"timestamp"`
}

// StartRequest represents start request payload.
type StartRequest struct {
	Port int `json:"port,omitempty"`
}

// StartResponse represents start response payload.
type StartResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Port    int    `json:"port"`
}

// ErrorResponse represents error response payload.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RegisterRoutes registers cursor-web API routes.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/cursor-web/status", handleStatus)
	mux.HandleFunc("POST /api/cursor-web/start", handleStart)
	mux.HandleFunc("POST /api/cursor-web/stop", handleStop)
	mux.HandleFunc("GET /api/cursor-web/status-stream", handleStatusStream)
	mux.HandleFunc("POST /api/cursor-web/start-stream", handleStartStream)
	mux.HandleFunc("POST /api/cursor-web/stop-stream", handleStopStream)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	manager := GetGlobalManager()

	running := manager.IsRunning() || manager.CheckServerHTTP()
	response := StatusResponse{
		Running:   running,
		Port:      manager.GetPort(),
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Port = 0
	}

	manager := GetGlobalManager()

	if manager.IsRunning() || manager.CheckServerHTTP() {
		response := StartResponse{
			Success: true,
			Message: "Cursor Web server is already running",
			Port:    manager.GetPort(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	if err := manager.Start(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := ErrorResponse{
			Error:   "failed_to_start",
			Message: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	time.Sleep(2 * time.Second)
	response := StartResponse{
		Success: true,
		Message: "Cursor Web server started successfully",
		Port:    manager.GetPort(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
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
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Cursor Web server stopped successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func handleStatusStream(w http.ResponseWriter, r *http.Request) {
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	manager := GetGlobalManager()
	sw.SendLog("Checking Cursor Web server status...")

	running := manager.IsRunning() || manager.CheckServerHTTP()
	port := manager.GetPort()
	if running {
		sw.SendLog(fmt.Sprintf("Cursor Web server is running on port %d", port))
	} else {
		sw.SendLog(fmt.Sprintf("Cursor Web server is not running on port %d", port))
	}

	sw.SendDone(map[string]string{
		"success": "true",
		"message": "Status check completed",
		"running": strconv.FormatBool(running),
		"port":    strconv.Itoa(port),
	})
}

func handleStartStream(w http.ResponseWriter, r *http.Request) {
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	manager := GetGlobalManager()
	port := manager.GetPort()
	sw.SendLog(fmt.Sprintf("Preparing to start Cursor Web server on port %d...", port))

	if manager.IsRunning() || manager.CheckServerHTTP() {
		sw.SendLog("Cursor Web server is already running")
		sw.SendDone(map[string]string{
			"success": "true",
			"message": "Cursor Web server is already running",
			"running": "true",
			"port":    strconv.Itoa(port),
		})
		return
	}

	sw.SendLog("Running: npx @siteboon/claude-code-ui --port " + strconv.Itoa(port))
	if err := manager.Start(); err != nil {
		sw.SendError("Failed to start Cursor Web server: " + err.Error())
		sw.SendDone(map[string]string{
			"success": "false",
			"message": err.Error(),
			"running": "false",
			"port":    strconv.Itoa(port),
		})
		return
	}

	sw.SendLog("Start command launched, waiting for server readiness...")
	time.Sleep(2 * time.Second)

	running := manager.IsRunning() || manager.CheckServerHTTP()
	if running {
		sw.SendLog(fmt.Sprintf("Cursor Web server started on port %d", port))
		sw.SendDone(map[string]string{
			"success": "true",
			"message": "Cursor Web server started successfully",
			"running": "true",
			"port":    strconv.Itoa(port),
		})
		return
	}

	sw.SendError("Start command finished, but server is not reachable yet")
	sw.SendDone(map[string]string{
		"success": "false",
		"message": "Cursor Web server is not reachable after start",
		"running": "false",
		"port":    strconv.Itoa(port),
	})
}

func handleStopStream(w http.ResponseWriter, r *http.Request) {
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	manager := GetGlobalManager()
	port := manager.GetPort()
	sw.SendLog(fmt.Sprintf("Preparing to stop Cursor Web server on port %d...", port))

	if !manager.IsRunning() && !manager.CheckServerHTTP() {
		sw.SendLog("Cursor Web server is already stopped")
		sw.SendDone(map[string]string{
			"success": "true",
			"message": "Cursor Web server is already stopped",
			"running": "false",
			"port":    strconv.Itoa(port),
		})
		return
	}

	if err := manager.Stop(); err != nil {
		sw.SendError("Failed to stop Cursor Web server: " + err.Error())
		sw.SendDone(map[string]string{
			"success": "false",
			"message": err.Error(),
			"running": "true",
			"port":    strconv.Itoa(port),
		})
		return
	}

	time.Sleep(500 * time.Millisecond)
	running := manager.IsRunning() || manager.CheckServerHTTP()
	if running {
		sw.SendError("Stop command returned, but server still appears running")
		sw.SendDone(map[string]string{
			"success": "false",
			"message": "Cursor Web server still running after stop",
			"running": "true",
			"port":    strconv.Itoa(port),
		})
		return
	}

	sw.SendLog("Cursor Web server stopped")
	sw.SendDone(map[string]string{
		"success": "true",
		"message": "Cursor Web server stopped successfully",
		"running": "false",
		"port":    strconv.Itoa(port),
	})
}
