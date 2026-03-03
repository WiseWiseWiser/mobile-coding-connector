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
	config := ConfigResponse{
		EnableMockupInMenu: os.Getenv(env.EnvEnableMockupInMenu) == "true",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
