package config

import (
	"encoding/json"
	"net/http"
	"os"
)

// ConfigResponse represents the server configuration exposed to the frontend
type ConfigResponse struct {
	EnableMockupInMenu bool `json:"enableMockupInMenu"`
}

// Handler returns the server configuration
func Handler(w http.ResponseWriter, r *http.Request) {
	config := ConfigResponse{
		EnableMockupInMenu: os.Getenv("ENABLE_MOCKUP_IN_MENU") == "true",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
