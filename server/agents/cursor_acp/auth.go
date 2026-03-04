package cursor_acp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type cursorAuth struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	APIKey       string `json:"apiKey,omitempty"`
}

func cursorAuthPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cursor", "auth.json")
}

func readCursorAuth() (*cursorAuth, error) {
	data, err := os.ReadFile(cursorAuthPath())
	if err != nil {
		return nil, err
	}
	var auth cursorAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

// ensureAuth checks that cursor-agent has valid authentication tokens.
// If ~/.config/cursor/auth.json is missing, it bootstraps auth by exchanging
// the configured API key for JWT tokens via a minimal cursor-agent invocation.
func ensureAuth(agentPath string) error {
	auth, err := readCursorAuth()
	if err == nil && auth.AccessToken != "" {
		return nil
	}

	settings, _ := LoadSettings()
	if settings == nil || settings.APIKey == "" {
		return fmt.Errorf("cursor-agent not authenticated and no API key configured; set an API key in settings or run 'cursor-agent login'")
	}

	if err := bootstrapAuth(agentPath, settings.APIKey); err != nil {
		return fmt.Errorf("failed to bootstrap auth: %w", err)
	}

	auth, err = readCursorAuth()
	if err != nil {
		return fmt.Errorf("auth.json not created after bootstrap: %w", err)
	}
	if auth.AccessToken == "" {
		return fmt.Errorf("auth.json missing access token after bootstrap")
	}
	if auth.APIKey != settings.APIKey {
		return fmt.Errorf("auth.json API key mismatch after bootstrap")
	}
	return nil
}

type LogFunc func(message string)

// validateAPIKey checks if a given API key is valid by bootstrapping auth
// with it and verifying the resulting auth.json.
// It temporarily backs up the existing auth.json to avoid corrupting a working key.
func validateAPIKey(agentPath, apiKey string, log LogFunc) error {
	if log == nil {
		log = func(string) {}
	}

	authPath := cursorAuthPath()
	backupPath := authPath + ".orig"

	_, existsErr := os.Stat(authPath)
	hasOrig := existsErr == nil
	if hasOrig {
		log("Backing up existing auth.json...")
		if err := os.Rename(authPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup auth.json: %w", err)
		}
	}
	defer func() {
		if hasOrig {
			log("Restoring original auth.json...")
			os.Remove(authPath)
			os.Rename(backupPath, authPath)
		}
	}()

	log("Running cursor-agent to validate API key...")
	if err := bootstrapAuth(agentPath, apiKey); err != nil {
		return fmt.Errorf("failed to bootstrap auth: %w", err)
	}

	log("Reading auth response...")
	auth, err := readCursorAuth()
	if err != nil {
		return fmt.Errorf("auth.json not created after validation: %w", err)
	}
	if auth.AccessToken == "" {
		return fmt.Errorf("auth.json missing access token; API key may be invalid")
	}
	if auth.APIKey != apiKey {
		return fmt.Errorf("auth.json API key mismatch")
	}
	return nil
}

// bootstrapAuth triggers cursor-agent's auth token exchange by running
// a minimal --print command with empty stdin. This creates ~/.config/cursor/auth.json
// with JWT tokens derived from the API key, which is required for the models subcommand.
// The command fails with "No prompt provided" but still performs the token exchange.
func bootstrapAuth(agentPath, apiKey string) error {
	cmd := exec.Command(agentPath,
		"--api-key", apiKey,
		"--print", "--output-format", "stream-json", "--yolo",
	)
	cmd.Stdin = strings.NewReader("")
	_ = cmd.Run()
	return nil
}
