## Preconditions

1. A temporary config home directory exists
2. `AI_CRITIC_HOME` is set to the temp directory
3. `opencode.json` is written to the config home with:
   - `WebServer.Enabled = true`
   - `WebServer.Port = 14096`
   - `DefaultDomain = "test-auto-start.example.com"`
   - `AuthProxyEnabled = false`
4. The server binary is built (or `go run` is available)
5. The server is started in normal (non-quick-test) mode on the configured port

## Steps

1. Validate `Request.OpenCodeSettings` is not nil and has required fields
2. Create a temporary directory for the config home
3. Set `AI_CRITIC_HOME` environment variable pointing to the temp directory
4. Write `opencode.json` with settings: `{"default_domain":"<domain>","web_server":{"enabled":true,"port":<port>,"auth_proxy_enabled":false}}`
5. Ensure the temp directory is created before writing the file
6. Log the written settings for debugging
7. Store the temp directory path for cleanup in the response

## Context

This leaf tests the happy path where opencode settings are configured to enable
auto-start with a valid external domain. The server should automatically trigger
`AutoStartWebServer()` during `RunStartupTasks()`.

The expected behaviour is:
- Server log contains `[opencode] AutoStartWebServer: BEGIN`
- Server log contains `[opencode] AutoStartWebServer: loaded settings`
- If the `opencode` binary is available: the web server starts on port 14096
- If the `opencode` binary is NOT available: `StartWebProcess` fails and the
  failure is logged (the auto-start mechanism itself still triggered)

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if req.OpenCodeSettings == nil {
		req.OpenCodeSettings = &OpenCodeSettings{
			WebServerEnabled: true,
			WebServerPort:    14096,
			DefaultDomain:    "test-auto-start.example.com",
		}
	}
	if req.OpenCodeSettings.DefaultDomain == "" {
		return fmt.Errorf("DefaultDomain must be set for auto-start test")
	}
	if !req.OpenCodeSettings.WebServerEnabled {
		return fmt.Errorf("WebServerEnabled must be true for auto-start test")
	}

	configHome := os.Getenv("AI_CRITIC_HOME")
	if configHome == "" {
		var err error
		configHome, err = os.MkdirTemp("", "ai-critic-test-*")
		if err != nil {
			return fmt.Errorf("failed to create temp config home: %w", err)
		}
		t.Logf("created config home: %s", configHome)
		t.Cleanup(func() {
			os.RemoveAll(configHome)
		})
		os.Setenv("AI_CRITIC_HOME", configHome)
		t.Cleanup(func() {
			os.Unsetenv("AI_CRITIC_HOME")
		})
	}

	if err := os.MkdirAll(configHome, 0755); err != nil {
		return fmt.Errorf("failed to ensure config home dir: %w", err)
	}

	webServerPort := req.OpenCodeSettings.WebServerPort
	if webServerPort <= 0 {
		webServerPort = 14096
	}

	settings := map[string]interface{}{
		"default_domain": req.OpenCodeSettings.DefaultDomain,
		"web_server": map[string]interface{}{
			"enabled":            req.OpenCodeSettings.WebServerEnabled,
			"port":               webServerPort,
			"auth_proxy_enabled": req.OpenCodeSettings.AuthProxyEnabled,
			"target_preference":  "domain",
		},
	}
	if req.OpenCodeSettings.BinaryPath != "" {
		settings["binary_path"] = req.OpenCodeSettings.BinaryPath
	}

	opencodeJSON := filepath.Join(configHome, "opencode.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode settings: %w", err)
	}
	if err := os.WriteFile(opencodeJSON, data, 0644); err != nil {
		return fmt.Errorf("failed to write opencode.json: %w", err)
	}
	t.Logf("wrote opencode.json to %s: %s", opencodeJSON, string(data))

	if req.ServerPort <= 0 {
		req.ServerPort = 23712
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 30
	}

	return nil
}
```
