package config

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/server/env"
)

// ConfigResponse represents the server configuration exposed to the frontend
type ConfigResponse struct {
	EnableMockupInMenu bool `json:"enableMockupInMenu"`
}

// Handler returns the server configuration
func Handler(w http.ResponseWriter, r *http.Request) {
	// Enable mockup in menu if either:
	// 1. ENABLE_MOCKUP_IN_MENU env var is set to "true"
	// 2. QUICK_TEST env var is set to "true" (quick-test mode)
	enableMockup := os.Getenv(env.EnvEnableMockupInMenu) == "true" || os.Getenv("QUICK_TEST") == "true"

	config := ConfigResponse{
		EnableMockupInMenu: enableMockup,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
