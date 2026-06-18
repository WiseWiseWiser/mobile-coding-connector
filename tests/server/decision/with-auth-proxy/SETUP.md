## Preconditions

1. A temporary config home directory exists
2. `AI_CRITIC_HOME` is set to the temp directory
3. `opencode.json` is written to the config home with:
   - `WebServer.Enabled = true`
   - `WebServer.Port = 14100`
   - `DefaultDomain = "test-auth-proxy.example.com"`
   - `AuthProxyEnabled = true`
4. The server binary and basic-auth-proxy binary are built
5. The server is started in normal (non-quick-test) mode on the configured port

## Steps

1. Validate `Request.OpenCodeSettings` is not nil and has required fields
2. Create a temporary directory for the config home
3. Set `AI_CRITIC_HOME` environment variable pointing to the temp directory
4. Write `opencode.json` with settings: `{"default_domain":"<domain>","web_server":{"enabled":true,"port":14100,"auth_proxy_enabled":true}}`
5. Ensure the temp directory is created before writing the file
6. Log the written settings for debugging
7. The root `Run` function builds the basic-auth-proxy binary and prepends it to PATH
8. The server starts in normal mode; `AutoStartWebServer()` triggers, detects `AuthProxyEnabled=true`, and starts the proxy

## Context

This leaf tests the auth proxy path where `AuthProxyEnabled=true` in opencode settings.
The server should automatically trigger `AutoStartWebServer()` during `RunStartupTasks()`,
which detects the auth proxy setting and calls `startWebServerWithProxy()`.

The expected behaviour is:
- Server log contains `[opencode] AutoStartWebServer: BEGIN`
- Server log contains `[opencode] AutoStartWebServer: loaded settings - ... AuthProxyEnabled=true`
- Server log contains `[basic_auth_proxy] Proxy started on port`
- If the `opencode` binary is available: the web server starts on a random backend port,
  the proxy starts on port 14100, proxying to the backend
- If the `opencode` binary is NOT available: `AutoStartWebServer` still triggers (logs appear)
  but the proxy may not start

The port architecture with proxy:
- Port 14100 = proxy listening port (public-facing, configured via `WebServer.Port`)
- Backend port = opencode web server internal port (random, read from `basic-auth-proxy.json`)

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
			WebServerPort:    14100,
			DefaultDomain:    "test-auth-proxy.example.com",
			AuthProxyEnabled: true,
		}
	}
	if req.OpenCodeSettings.DefaultDomain == "" {
		return fmt.Errorf("DefaultDomain must be set for auto-start test")
	}
	if !req.OpenCodeSettings.WebServerEnabled {
		return fmt.Errorf("WebServerEnabled must be true for auto-start test")
	}
	if !req.OpenCodeSettings.AuthProxyEnabled {
		return fmt.Errorf("AuthProxyEnabled must be true for auth proxy test")
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

	settings := map[string]interface{}{
		"default_domain": req.OpenCodeSettings.DefaultDomain,
		"web_server": map[string]interface{}{
			"enabled":            req.OpenCodeSettings.WebServerEnabled,
			"port":               req.OpenCodeSettings.WebServerPort,
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
