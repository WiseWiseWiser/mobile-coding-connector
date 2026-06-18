## Preconditions

1. A temporary config home directory exists
2. `AI_CRITIC_HOME` is set to the temp directory
3. `opencode.json` is written to the config home with:
   - `WebServer.Enabled = false`
   - `WebServer.Port = 14096`
   - `DefaultDomain = "test-disabled.example.com"`
   - `AuthProxyEnabled = false`
4. The server binary is built
5. The server is started in normal (non-quick-test) mode

## Steps

1. Set `Request.OpenCodeSettings` with `WebServerEnabled=false`, a valid `DefaultDomain`, and `WebServerPort=14096`
2. Write `opencode.json` with `enabled: false` to the config home
3. Let the root `Run` function build and start the server
4. `RunStartupTasks()` should check `WebServer.Enabled` and skip `AutoStartWebServer()` when it is false

## Context

This leaf tests that when `WebServer.Enabled` is `false` in opencode settings,
the server does NOT trigger `AutoStartWebServer()` during startup. The server
should still start normally (respond to `/ping`), but the log should contain
no `[opencode] AutoStartWebServer:` messages.

The expected behaviour is:
- Server starts and responds to `/ping`
- Server log does NOT contain `[opencode] AutoStartWebServer:`
- The opencode web server port should NOT become accessible
```

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.OpenCodeSettings = &OpenCodeSettings{
		WebServerEnabled: false,
		WebServerPort:    14096,
		DefaultDomain:    "test-disabled.example.com",
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
			"enabled":            false,
			"port":               req.OpenCodeSettings.WebServerPort,
			"auth_proxy_enabled": false,
		},
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
