package quicktest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// RegisterQuickTestAPI registers quick-test specific endpoints
// These endpoints are only available when quick-test mode is enabled
func RegisterQuickTestAPI(mux *http.ServeMux) {
	// Health and status endpoints
	mux.HandleFunc("/api/quick-test/health", handleHealth)
	mux.HandleFunc("/api/quick-test/status", handleStatus)
	mux.HandleFunc("/api/quick-test/config", handleConfig)

	// Auth proxy testing endpoints
	mux.HandleFunc("/api/quick-test/auth-proxy/status", handleAuthProxyStatus)
	mux.HandleFunc("/api/quick-test/auth-proxy/start", handleAuthProxyStart)
	mux.HandleFunc("/api/quick-test/auth-proxy/stop", handleAuthProxyStop)

	// Webserver testing endpoints
	mux.HandleFunc("/api/quick-test/webserver/status", handleWebServerStatus)
	mux.HandleFunc("/api/quick-test/webserver/settings", handleWebServerSettings)

	// Environment and system info
	mux.HandleFunc("/api/quick-test/env", handleEnv)
	mux.HandleFunc("/api/quick-test/logs", handleLogs)

	// Auto-start webserver (to test the auth proxy startup issue)
	mux.HandleFunc("/api/quick-test/webserver/autostart", handleWebServerAutoStart)
}

// QuickTestLogger interface for logging
// This avoids direct imports that cause cycles
type QuickTestLogger interface {
	LogInfo(format string, args ...interface{})
	LogError(format string, args ...interface{})
}

// handleHealth returns a simple health check response
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"mode":      "quick-test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleStatus returns detailed server status
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"quick_test": map[string]interface{}{
			"enabled":      Enabled(),
			"keep_enabled": KeepEnabled(),
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleConfig returns quick-test configuration
func handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":             Enabled(),
		"keep_enabled":        KeepEnabled(),
		"exec_restart_binary": GetExecRestartBinary(),
	})
}

// handleAuthProxyStatus returns auth proxy status
// This uses a simple HTTP check to avoid import cycles
func handleAuthProxyStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Default response - we'll check via HTTP in the debug script
	json.NewEncoder(w).Encode(map[string]interface{}{
		"note":      "Use debug script to check actual auth proxy status",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleAuthProxyStart attempts to start the auth proxy
// The actual startup is delegated to the webserver control
func handleAuthProxyStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   "Auth proxy start requires webserver control. Use /api/quick-test/webserver/autostart",
	})
}

// handleAuthProxyStop attempts to stop the auth proxy
func handleAuthProxyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   "Auth proxy stop requires webserver control. Use /api/quick-test/webserver/autostart",
	})
}

// handleWebServerStatus returns webserver status placeholder
func handleWebServerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"note": "Webserver status requires settings storage. Check logs for autostart results.",
	})
}

// handleWebServerSettings returns webserver settings placeholder
func handleWebServerSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"note": "Webserver settings require settings storage.",
	})
}

// AutostartCallback is a function type that triggers the webserver autostart
var AutostartCallback func()

// handleWebServerAutoStart triggers the autostart sequence
func handleWebServerAutoStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if AutostartCallback != nil {
		// Call the actual autostart function
		go AutostartCallback()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Autostart triggered. Check server logs for results.",
			"note":    "Autostart callback invoked - check logs for webserver startup results",
		})
		return
	}

	// No callback registered - this is expected in quick-test mode
	// because RunSideEffectTasks is not called
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": "Autostart callback not registered",
		"note":    "In quick-test mode, RunSideEffectTasks is skipped, so AutoStartWebServer is not called. This endpoint allows manual triggering for debugging.",
	})
}

// handleEnv returns environment information
func handleEnv(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get relevant environment variables
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "OPENCODE_") ||
			strings.HasPrefix(env, "AI_CRITIC_") ||
			strings.HasPrefix(env, "HOME") ||
			strings.HasPrefix(env, "USER") ||
			strings.HasPrefix(env, "PATH") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envVars[parts[0]] = parts[1]
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"num_cpu":    runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"env_vars":   envVars,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleLogs returns recent log entries (placeholder)
func handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// For now, return a placeholder
	// In a real implementation, this would read from a log buffer or file
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": []map[string]string{
			{"level": "info", "message": "Quick-test mode enabled", "timestamp": time.Now().UTC().Format(time.RFC3339)},
		},
		"note": "This is a placeholder. Full log streaming would require implementing a log buffer.",
	})
}

// LogInfo logs a message to the quick-test log
// This can be used throughout the codebase to log quick-test specific information
func LogInfo(format string, args ...interface{}) {
	if Enabled() {
		fmt.Printf("[quick-test] "+format+"\n", args...)
	}
}

// LogError logs an error to the quick-test log
func LogError(format string, args ...interface{}) {
	if Enabled() {
		fmt.Fprintf(os.Stderr, "[quick-test] ERROR: "+format+"\n", args...)
	}
}
